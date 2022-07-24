package client

import (
	"context"
	"errors"
)

// selects a client from candidates base on c.selectMode
func (c *xClient) selectClient(ctx context.Context, servicePath, serviceMethod string, args interface{}) (string, RPCClient, error) {
	c.mu.Lock()
	fn := c.selector.Select
	if c.Plugins != nil {
		fn = c.Plugins.DoWrapSelect(fn)
	}
	k := fn(ctx, servicePath, serviceMethod, args)
	c.mu.Unlock()
	if k == "" {
		return "", nil, ErrXClientNoServer
	}
	client, err := c.getCachedClient(k, servicePath, serviceMethod, args)
	return k, client, err
}

func (c *xClient) getCachedClient(k string, servicePath, serviceMethod string, _ interface{}) (RPCClient, error) {
	// TODO: improve the lock
	var client RPCClient
	var needCallPlugin bool
	defer func() {
		if needCallPlugin {
			_, _ = c.Plugins.DoClientConnected(client.GetConn())
		}
	}()

	if c.isShutdown {
		return nil, errors.New("this xClient is closed")
	}

	// if this client is broken
	//breaker, ok := c.breakers.Load(k)
	if breaker, ok := c.breakers.Load(k); ok && !breaker.(Breaker).Ready() {
		return nil, ErrBreakerOpen
	}

	c.mu.Lock()
	client = c.findCachedClient(k, servicePath, serviceMethod)
	if client != nil {
		if !client.IsClosing() && !client.IsShutdown() {
			c.mu.Unlock()
			return client, nil
		}
		c.deleteCachedClient(client, k, servicePath, serviceMethod)
	}

	client = c.findCachedClient(k, servicePath, serviceMethod)
	c.mu.Unlock()

	if client == nil || client.IsShutdown() {
		c.mu.Lock()
		generatedClient, err, _ := c.slGroup.Do(k, func() (interface{}, error) {
			return c.generateClient(k, servicePath, serviceMethod)
		})
		c.mu.Unlock()

		c.slGroup.Forget(k)
		if err != nil {
			return nil, err
		}

		client = generatedClient.(RPCClient)
		if c.Plugins != nil {
			needCallPlugin = true
		}

		client.RegisterServerMessageChan(c.serverMessageChan)

		c.mu.Lock()
		c.setCachedClient(client, k, servicePath, serviceMethod)
		c.mu.Unlock()
	}

	return client, nil
}

func (c *xClient) findCachedClient(k, servicePath, serviceMethod string) RPCClient {
	network, _ := splitNetworkAndAddress(k)
	if builder, ok := getCacheClientBuilder(network); ok {
		return builder.FindCachedClient(k, servicePath, serviceMethod)
	}

	return c.cachedClient[k]
}

func (c *xClient) deleteCachedClient(client RPCClient, k, servicePath, serviceMethod string) {
	network, _ := splitNetworkAndAddress(k)
	if builder, ok := getCacheClientBuilder(network); ok && client != nil {
		builder.DeleteCachedClient(client, k, servicePath, serviceMethod)
		_ = client.Close()
		return
	}

	delete(c.cachedClient, k)
	if client != nil {
		_ = client.Close()
	}
}

func (c *xClient) generateClient(k, servicePath, serviceMethod string) (client RPCClient, err error) {
	network, addr := splitNetworkAndAddress(k)
	if builder, ok := getCacheClientBuilder(network); ok && builder != nil {
		return builder.GenerateClient(k, servicePath, serviceMethod)
	}

	client = &Client{
		option:  c.option,
		Plugins: c.Plugins,
	}

	var breaker interface{}
	if c.option.GenBreaker != nil {
		breaker, _ = c.breakers.LoadOrStore(k, c.option.GenBreaker())
	}

	err = client.Connect(network, addr)
	if err != nil {
		if breaker != nil {
			breaker.(Breaker).Fail()
		}
		return nil, err
	}
	return client, err
}

func (c *xClient) setCachedClient(client RPCClient, k, servicePath, serviceMethod string) {
	network, _ := splitNetworkAndAddress(k)
	if builder, ok := getCacheClientBuilder(network); ok {
		builder.SetCachedClient(client, k, servicePath, serviceMethod)
		return
	}

	c.cachedClient[k] = client
}

func (c *xClient) removeClient(k, servicePath, serviceMethod string, client RPCClient) {
	c.mu.Lock()
	cl := c.findCachedClient(k, servicePath, serviceMethod)
	if cl == client {
		c.deleteCachedClient(client, k, servicePath, serviceMethod)
	}
	c.mu.Unlock()

	if client != nil {
		client.UnregisterServerMessageChan()
		_ = client.Close()
	}
}

func (c *xClient) getCachedClientWithoutLock(k, servicePath, serviceMethod string) (RPCClient, bool, error) {
	var needCallPlugin bool
	client := c.findCachedClient(k, servicePath, serviceMethod)
	if client != nil {
		if !client.IsClosing() && !client.IsShutdown() {
			return client, needCallPlugin, nil
		}
		c.deleteCachedClient(client, k, servicePath, serviceMethod)
	}

	// double check
	client = c.findCachedClient(k, servicePath, serviceMethod)
	if client == nil || client.IsShutdown() {
		generatedClient, err, _ := c.slGroup.Do(k, func() (interface{}, error) {
			return c.generateClient(k, servicePath, serviceMethod)
		})
		c.slGroup.Forget(k)
		if err != nil {
			return nil, needCallPlugin, err
		}

		client = generatedClient.(RPCClient)
		if c.Plugins != nil {
			needCallPlugin = true
		}

		client.RegisterServerMessageChan(c.serverMessageChan)

		c.setCachedClient(client, k, servicePath, serviceMethod)
	}

	return client, needCallPlugin, nil
}
