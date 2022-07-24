package client

import (
	"context"
	"github.com/derekAHua/irpc/protocol"
	"net"
)

type (
	// PluginContainer represents a plugin container that defines all methods to manage plugins.
	// And it also defines all extension points.
	PluginContainer interface {
		// Add adds some plugins.
		Add(plugins ...Plugin)
		Remove(plugin Plugin)
		All() []Plugin

		DoConnCreated(net.Conn) (net.Conn, error)
		DoConnCreateFailed(network, address string)
		DoClientConnected(net.Conn) (net.Conn, error)
		DoClientConnectionClose(net.Conn) error

		DoPreCall(ctx context.Context, servicePath, serviceMethod string, args interface{}) error
		DoPostCall(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, err error) error

		// DoClientBeforeEncode is called when requests are encoded and sent.
		DoClientBeforeEncode(*protocol.Message) error
		// DoClientAfterDecode is called after requests are decoded.
		DoClientAfterDecode(*protocol.Message) error

		// DoWrapSelect is called when select a node.
		DoWrapSelect(SelectFunc) SelectFunc
	}
)

// pluginContainer implements PluginContainer interface.
type pluginContainer struct {
	plugins []Plugin
}

func NewPluginContainer() PluginContainer {
	return &pluginContainer{}
}

func (p *pluginContainer) Add(plugins ...Plugin) {
	p.plugins = append(p.plugins, plugins...)
}

// Remove removes a plugin by its name.
func (p *pluginContainer) Remove(plugin Plugin) {
	if p.plugins == nil {
		return
	}

	var plugins []Plugin
	for _, pp := range p.plugins {
		if pp != plugin {
			plugins = append(plugins, pp)
		}
	}

	p.plugins = plugins
}

// All returns all plugins
func (p *pluginContainer) All() []Plugin {
	return p.plugins
}

// DoPreCall executes before call
func (p *pluginContainer) DoPreCall(ctx context.Context, servicePath, serviceMethod string, args interface{}) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreCallPlugin); ok {
			err := plugin.PreCall(ctx, servicePath, serviceMethod, args)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPostCall executes after call
func (p *pluginContainer) DoPostCall(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, err error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostCallPlugin); ok {
			err = plugin.PostCall(ctx, servicePath, serviceMethod, args, reply, err)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DoConnCreated is called in case of client connection created.
func (p *pluginContainer) DoConnCreated(conn net.Conn) (net.Conn, error) {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(ConnCreatedPlugin); ok {
			conn, err = plugin.ConnCreated(conn)
			if err != nil {
				return conn, err
			}
		}
	}
	return conn, nil
}

// DoConnCreateFailed is called in case of client connection created.
func (p *pluginContainer) DoConnCreateFailed(network, address string) {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(ConnCreateFailedPlugin); ok {
			plugin.ConnCreateFailed(network, address)
		}
	}
}

// DoClientConnected is called in case of connected.
func (p *pluginContainer) DoClientConnected(conn net.Conn) (net.Conn, error) {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(ConnectedPlugin); ok {
			conn, err = plugin.ClientConnected(conn)
			if err != nil {
				return conn, err
			}
		}
	}
	return conn, nil
}

// DoClientConnectionClose is called in case of connection closed.
func (p *pluginContainer) DoClientConnectionClose(conn net.Conn) error {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(ConnectionClosePlugin); ok {
			err = plugin.ClientConnectionClose(conn)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (p *pluginContainer) DoClientBeforeEncode(req *protocol.Message) error {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(BeforeEncodePlugin); ok {
			err = plugin.ClientBeforeEncode(req)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *pluginContainer) DoClientAfterDecode(req *protocol.Message) error {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(AfterDecodePlugin); ok {
			err = plugin.ClientAfterDecode(req)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *pluginContainer) DoWrapSelect(fn SelectFunc) SelectFunc {
	rt := fn
	for i := range p.plugins {
		if pn, ok := p.plugins[i].(SelectNodePlugin); ok {
			rt = pn.WrapSelect(rt)
		}
	}

	return rt
}
