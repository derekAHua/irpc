package server

import (
	"context"
	"github.com/derekAHua/irpc/errors"
	"github.com/derekAHua/irpc/protocol"
	"github.com/soheilhy/cmux"
	"net"
)

// PluginContainer represents a plugin container that defines all methods to manage plugins.
// And it also defines all extension points.
type PluginContainer interface {
	Add(plugins ...Plugin)
	Remove(plugin Plugin)
	All() []Plugin

	DoRegister(name string, receiver interface{}, metadata string) error
	DoRegisterFunction(serviceName, fname string, fn interface{}, metadata string) error
	DoUnregister(name string) error

	// DoPostConnAccept handles accepted conn.
	DoPostConnAccept(net.Conn) (net.Conn, bool)
	// DoPostConnClose handles closed conn.
	DoPostConnClose(net.Conn) bool

	DoPreReadRequest(ctx context.Context) error
	DoPostReadRequest(ctx context.Context, r *protocol.Message, e error) error

	DoPreHandleRequest(ctx context.Context, req *protocol.Message) error
	DoPreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error)
	DoPostCall(ctx context.Context, serviceName, methodName string, args, reply interface{}) (interface{}, error)

	DoPreWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error
	DoPostWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error

	DoPreWriteRequest(ctx context.Context) error
	DoPostWriteRequest(ctx context.Context, r *protocol.Message, e error) error

	DoHeartbeatRequest(ctx context.Context, req *protocol.Message) error

	MuxMatch(m cmux.CMux)
}

// pluginContainer implements PluginContainer interface.
type pluginContainer struct {
	plugins []Plugin
}

// Add adds some plugins.
func (p *pluginContainer) Add(plugins ...Plugin) {
	p.plugins = append(p.plugins, plugins...)
}

// Remove removes a plugin by its name.
func (p *pluginContainer) Remove(plugin Plugin) {
	if p.plugins == nil {
		return
	}

	plugins := make([]Plugin, 0, len(p.plugins))
	for _, p := range p.plugins {
		if p != plugin {
			plugins = append(plugins, p)
		}
	}

	p.plugins = plugins
}

func (p *pluginContainer) All() []Plugin {
	return p.plugins
}

// DoRegister invokes DoRegister plugin.
func (p *pluginContainer) DoRegister(name string, rcvr interface{}, metadata string) error {
	var es []error
	for _, rp := range p.plugins {
		if plugin, ok := rp.(RegisterPlugin); ok {
			err := plugin.Register(name, rcvr, metadata)
			if err != nil {
				es = append(es, err)
			}
		}
	}

	if len(es) > 0 {
		return errors.NewMultiError(es)
	}
	return nil
}

// DoRegisterFunction invokes DoRegisterFunction plugin.
func (p *pluginContainer) DoRegisterFunction(serviceName, fname string, fn interface{}, metadata string) error {
	var es []error
	for _, rp := range p.plugins {
		if plugin, ok := rp.(RegisterFunctionPlugin); ok {
			err := plugin.RegisterFunction(serviceName, fname, fn, metadata)
			if err != nil {
				es = append(es, err)
			}
		}
	}

	if len(es) > 0 {
		return errors.NewMultiError(es)
	}
	return nil
}

// DoUnregister invokes RegisterPlugin.
func (p *pluginContainer) DoUnregister(name string) error {
	var es []error
	for _, rp := range p.plugins {
		if plugin, ok := rp.(RegisterPlugin); ok {
			err := plugin.Unregister(name)
			if err != nil {
				es = append(es, err)
			}
		}
	}

	if len(es) > 0 {
		return errors.NewMultiError(es)
	}
	return nil
}

func (p *pluginContainer) DoPostConnAccept(conn net.Conn) (net.Conn, bool) {
	var flag bool
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostConnAcceptPlugin); ok {
			conn, flag = plugin.HandleConnAccept(conn)
			if !flag { // interrupt
				_ = conn.Close()
				return conn, false
			}
		}
	}
	return conn, true
}

func (p *pluginContainer) DoPostConnClose(conn net.Conn) bool {
	var flag bool
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostConnClosePlugin); ok {
			flag = plugin.HandleConnClose(conn)
			if !flag {
				return false
			}
		}
	}
	return true
}

// DoPreReadRequest invokes PreReadRequest plugin.
func (p *pluginContainer) DoPreReadRequest(ctx context.Context) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreReadRequestPlugin); ok {
			err := plugin.PreReadRequest(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPostReadRequest invokes PostReadRequest plugin.
func (p *pluginContainer) DoPostReadRequest(ctx context.Context, r *protocol.Message, e error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostReadRequestPlugin); ok {
			err := plugin.PostReadRequest(ctx, r, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPreHandleRequest invokes PreHandleRequest plugin.
func (p *pluginContainer) DoPreHandleRequest(ctx context.Context, r *protocol.Message) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreHandleRequestPlugin); ok {
			err := plugin.PreHandleRequest(ctx, r)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPreCall invokes PreCallPlugin plugin.
func (p *pluginContainer) DoPreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error) {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreCallPlugin); ok {
			args, err = plugin.PreCall(ctx, serviceName, methodName, args)
			if err != nil {
				return args, err
			}
		}
	}

	return args, err
}

// DoPostCall invokes PostCallPlugin plugin.
func (p *pluginContainer) DoPostCall(ctx context.Context, serviceName, methodName string, args, reply interface{}) (interface{}, error) {
	var err error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostCallPlugin); ok {
			reply, err = plugin.PostCall(ctx, serviceName, methodName, args, reply)
			if err != nil {
				return reply, err
			}
		}
	}

	return reply, err
}

// DoPreWriteResponse invokes PreWriteResponse plugin.
func (p *pluginContainer) DoPreWriteResponse(ctx context.Context, req *protocol.Message, res *protocol.Message, err error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreWriteResponsePlugin); ok {
			err := plugin.PreWriteResponse(ctx, req, res, err)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPostWriteResponse invokes PostWriteResponse plugin.
func (p *pluginContainer) DoPostWriteResponse(ctx context.Context, req *protocol.Message, resp *protocol.Message, e error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostWriteResponsePlugin); ok {
			err := plugin.PostWriteResponse(ctx, req, resp, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPreWriteRequest invokes PreWriteRequest plugin.
func (p *pluginContainer) DoPreWriteRequest(ctx context.Context) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreWriteRequestPlugin); ok {
			err := plugin.PreWriteRequest(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPostWriteRequest invokes PostWriteRequest plugin.
func (p *pluginContainer) DoPostWriteRequest(ctx context.Context, r *protocol.Message, e error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostWriteRequestPlugin); ok {
			err := plugin.PostWriteRequest(ctx, r, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoHeartbeatRequest invokes HeartbeatRequest plugin.
func (p *pluginContainer) DoHeartbeatRequest(ctx context.Context, r *protocol.Message) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(HeartbeatPlugin); ok {
			err := plugin.HeartbeatRequest(ctx, r)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// MuxMatch adds cmux.CMux Match.
func (p *pluginContainer) MuxMatch(m cmux.CMux) {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(CMuxPlugin); ok {
			plugin.MuxMatch(m)
		}
	}
}
