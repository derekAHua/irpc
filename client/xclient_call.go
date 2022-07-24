package client

import (
	"context"
	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/share"
	"reflect"
	"time"
)

// Call invokes the named function, waits for it to complete, and returns its error status.
// It handles errors base on FailMode.
func (c *xClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	if c.isShutdown {
		return ErrXClientShutdown
	}

	if c.auth != "" {
		metadata := ctx.Value(share.ReqMetaDataKey)
		if metadata == nil {
			metadata = map[string]string{}
			ctx = context.WithValue(ctx, share.ReqMetaDataKey, metadata)
		}
		m := metadata.(map[string]string)
		m[share.AuthKey] = c.auth
	}
	ctx = setServerTimeout(ctx)

	if share.Trace {
		log.Debugf("select a client for %s.%s, failMode: %v, args: %+v in case of xclient Call", c.servicePath, serviceMethod, c.failMode, args)
	}

	var err error
	k, client, err := c.selectClient(ctx, c.servicePath, serviceMethod, args)
	if err != nil {
		if c.failMode == Failfast || contextCanceled(err) {
			return err
		}
	}

	if share.Trace {
		if client != nil {
			log.Debugf("selected a client %s for %s.%s, failMode: %v, args: %+v in case of xClient Call", client.RemoteAddr(), c.servicePath, serviceMethod, c.failMode, args)
		} else {
			log.Debugf("selected a client %s for %s.%s, failMode: %v, args: %+v in case of xClient Call", "nil", c.servicePath, serviceMethod, c.failMode, args)
		}
	}

	var e error
	switch c.failMode {
	case Failtry:
		retries := c.option.Retries
		for retries >= 0 {
			retries--

			if client != nil {
				err = c.wrapCall(ctx, client, serviceMethod, args, reply)
				if err == nil {
					return nil
				}
				if contextCanceled(err) {
					return err
				}
				if _, ok := err.(ServiceError); ok {
					return err
				}
			}

			if uncoverError(err) {
				c.removeClient(k, c.servicePath, serviceMethod, client)
			}
			client, e = c.getCachedClient(k, c.servicePath, serviceMethod, args)
		}
		if err == nil {
			err = e
		}
		return err
	case Failover:
		retries := c.option.Retries
		for retries >= 0 {
			retries--

			if client != nil {
				err = c.wrapCall(ctx, client, serviceMethod, args, reply)
				if err == nil {
					return nil
				}
				if contextCanceled(err) {
					return err
				}
				if _, ok := err.(ServiceError); ok {
					return err
				}
			}

			if uncoverError(err) {
				c.removeClient(k, c.servicePath, serviceMethod, client)
			}
			// select another server
			k, client, e = c.selectClient(ctx, c.servicePath, serviceMethod, args)
		}

		if err == nil {
			err = e
		}
		return err
	case Failbackup:
		ctx, cancelFn := context.WithCancel(ctx)
		defer cancelFn()
		call1 := make(chan *Call, 10)
		call2 := make(chan *Call, 10)

		var reply1, reply2 interface{}

		if reply != nil {
			reply1 = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			reply2 = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
		}

		_, err1 := c.Go(ctx, serviceMethod, args, reply1, call1)

		t := time.NewTimer(c.option.BackupLatency)
		select {
		case <-ctx.Done(): // cancel by context
			err = ctx.Err()
			return err
		case call := <-call1:
			err = call.Error
			if err == nil && reply != nil {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(reply1).Elem())
			}
			return err
		case <-t.C:
		}

		_, err2 := c.Go(ctx, serviceMethod, args, reply2, call2)
		if err2 != nil {
			if uncoverError(err2) {
				c.removeClient(k, c.servicePath, serviceMethod, client)
			}
			err = err1
			return err
		}

		select {
		case <-ctx.Done(): // cancel by context
			err = ctx.Err()
		case call := <-call1:
			err = call.Error
			if err == nil && reply != nil && reply1 != nil {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(reply1).Elem())
			}
		case call := <-call2:
			err = call.Error
			if err == nil && reply != nil && reply2 != nil {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(reply2).Elem())
			}
		}

		return err
	default: // FailFast
		err = c.wrapCall(ctx, client, serviceMethod, args, reply)
		if err != nil {
			if uncoverError(err) {
				c.removeClient(k, c.servicePath, serviceMethod, client)
			}
		}

		return err
	}
}

func (c *xClient) wrapCall(ctx context.Context, client RPCClient, serviceMethod string, args interface{}, reply interface{}) error {
	if client == nil {
		return ErrServerUnavailable
	}

	if share.Trace {
		log.Debugf("call a client for %s.%s, args: %+v in case of xclient wrapCall", c.servicePath, serviceMethod, args)
	}

	ctx = share.NewContext(ctx)

	_ = c.Plugins.DoPreCall(ctx, c.servicePath, serviceMethod, args)
	err := client.Call(ctx, c.servicePath, serviceMethod, args, reply)
	_ = c.Plugins.DoPostCall(ctx, c.servicePath, serviceMethod, args, reply, err)

	if share.Trace {
		log.Debugf("called a client for %s.%s, args: %+v, err: %v in case of xclient wrapCall", c.servicePath, serviceMethod, args, err)
	}

	return err
}

// Go invokes the function asynchronously. It returns the Call structure representing the invocation.
// The done channel will signal when the call is complete by returning the same Call object.
// If done is nil, Go will allocate a new channel. If non-nil, done must be buffered or Go will deliberately crash.
// It does not use FailMode.
func (c *xClient) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error) {
	if c.isShutdown {
		return nil, ErrXClientShutdown
	}

	if c.auth != "" {
		metadata := ctx.Value(share.ReqMetaDataKey)
		if metadata == nil {
			metadata = map[string]string{}
			ctx = context.WithValue(ctx, share.ReqMetaDataKey, metadata)
		}
		m := metadata.(map[string]string)
		m[share.AuthKey] = c.auth
	}

	ctx = setServerTimeout(ctx)

	if share.Trace {
		log.Debugf("select a client for %s.%s, args: %+v in case of xclient Go", c.servicePath, serviceMethod, args)
	}
	_, client, err := c.selectClient(ctx, c.servicePath, serviceMethod, args)
	if err != nil {
		return nil, err
	}
	if share.Trace {
		log.Debugf("selected a client %s for %s.%s, args: %+v in case of xclient Go", client.RemoteAddr(), c.servicePath, serviceMethod, args)
	}
	return client.Go(ctx, c.servicePath, serviceMethod, args, reply, done), nil
}

func uncoverError(err error) bool {
	if _, ok := err.(ServiceError); ok {
		return false
	}

	if err == context.DeadlineExceeded {
		return false
	}

	if err == context.Canceled {
		return false
	}

	return true
}
