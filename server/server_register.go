package server

import (
	"errors"
	"fmt"
	"github.com/derekAHua/irpc/log"
	"reflect"
	"runtime"
	"strings"
)

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//	- exported method of exported type
//	- three arguments, the first is of context.Context, both of exported type for three arguments
//	- the third argument is a pointer
//	- one return value, of type error
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (s *Server) Register(receiver interface{}, metadata string) error {
	serviceName, err := s.register(receiver, "", false)
	if err != nil {
		return err
	}
	return s.Plugins.DoRegister(serviceName, receiver, metadata)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (s *Server) RegisterName(name string, receiver interface{}, metadata string) error {
	_, err := s.register(receiver, name, true)
	if err != nil {
		return err
	}
	if s.Plugins == nil {
		s.Plugins = &pluginContainer{}
	}
	return s.Plugins.DoRegister(name, receiver, metadata)
}

// RegisterFunction publishes a function that satisfy the following conditions:
//	- three arguments, the first is of context.Context, both of exported type for three arguments
//	- the third argument is a pointer
//	- one return value, of type error
// The client accesses function using a string of the form "servicePath.Method".
func (s *Server) RegisterFunction(servicePath string, fn interface{}, metadata string) error {
	serviceName, err := s.registerFunction(servicePath, fn, "", false)
	if err != nil {
		return err
	}

	return s.Plugins.DoRegisterFunction(servicePath, serviceName, fn, metadata)
}

// RegisterFunctionName is like RegisterFunction but uses the provided name for the function
// instead of the function's concrete type.
func (s *Server) RegisterFunctionName(servicePath string, name string, fn interface{}, metadata string) error {
	_, err := s.registerFunction(servicePath, fn, name, true)
	if err != nil {
		return err
	}

	return s.Plugins.DoRegisterFunction(servicePath, name, fn, metadata)
}

func (s *Server) register(receiver interface{}, name string, useName bool) (serviceName string, err error) {
	service := new(service)
	service.typ = reflect.TypeOf(receiver)
	service.receiver = reflect.ValueOf(receiver)
	serviceName = reflect.Indirect(service.receiver).Type().Name()
	if useName {
		serviceName = name
	}

	err = checkServiceName(serviceName, useName)
	if err != nil {
		return
	}

	service.name = serviceName

	// Install the methods
	service.method = suitableMethods(service.typ, true)

	if len(service.method) == 0 {
		errStr := "irpc.Register: type " + serviceName + " has no exported methods of suitable type"

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(service.typ), false)
		if len(method) != 0 {
			errStr = "irpc.Register: type " + serviceName + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		}

		log.Error(errStr)
		return serviceName, errors.New(errStr)
	}

	s.serviceMapMu.Lock()
	defer s.serviceMapMu.Unlock()
	s.serviceMap[service.name] = service

	return
}

func checkServiceName(serviceName string, useName bool) (err error) {
	if serviceName == "" {
		errStr := "irpc.checkServiceName: serviceName should be set"
		log.Error(errStr)
		err = errors.New(errStr)
		return
	}

	if !useName && !isExported(serviceName) {
		errStr := "irpc.checkServiceName: type " + serviceName + " is not exported."
		log.Error(errStr)
		err = errors.New(errStr)
		return
	}

	return
}

func (s *Server) registerFunction(servicePath string, fn interface{}, name string, useName bool) (fName string, err error) {
	s.serviceMapMu.Lock()
	defer s.serviceMapMu.Unlock()

	ss := s.serviceMap[servicePath]
	if ss == nil {
		ss = new(service)
		ss.name = servicePath
		ss.function = make(map[string]*functionType)
	}

	f, ok := fn.(reflect.Value)
	if !ok {
		f = reflect.ValueOf(fn)
	}
	if f.Kind() != reflect.Func {
		err = errors.New("function must be func or bound method")
		return
	}

	fName = runtime.FuncForPC(reflect.Indirect(f).Pointer()).Name()
	if fName != "" {
		i := strings.LastIndex(fName, ".")
		if i >= 0 {
			fName = fName[i+1:]
		}
	}
	if useName {
		fName = name
	}
	if fName == "" {
		errorStr := "irpc.registerFunction: no func name for type " + f.Type().String()
		log.Error(errorStr)
		return fName, errors.New(errorStr)
	}

	t := f.Type()
	if t.NumIn() != 3 {
		return fName, fmt.Errorf("irpc.registerFunction: has wrong number of ins: %s", f.Type().String())
	}
	if t.NumOut() != 1 {
		return fName, fmt.Errorf("irpc.registerFunction: has wrong number of outs: %s", f.Type().String())
	}

	// First arg must be context.Context
	ctxType := t.In(0)
	if !ctxType.Implements(typeOfContext) {
		return fName, fmt.Errorf("function %s must use context as  the first parameter", f.Type().String())
	}

	argType := t.In(1)
	if !isExportedOrBuiltinType(argType) {
		return fName, fmt.Errorf("function %s parameter type not exported: %v", f.Type().String(), argType)
	}

	replyType := t.In(2)
	if replyType.Kind() != reflect.Ptr {
		return fName, fmt.Errorf("function %s reply type not a pointer: %s", f.Type().String(), replyType)
	}
	if !isExportedOrBuiltinType(replyType) {
		return fName, fmt.Errorf("function %s reply type not exported: %v", f.Type().String(), replyType)
	}

	// The return type of the method must be error.
	if returnType := t.Out(0); returnType != typeOfError {
		return fName, fmt.Errorf("function %s returns %s, not error", f.Type().String(), returnType.String())
	}

	// Install the methods
	ss.function[fName] = &functionType{fn: f, ArgType: argType, ReplyType: replyType}
	s.serviceMap[servicePath] = ss

	// init pool for reflect.Type of args and reply
	reflectTypePools.Init(argType)
	reflectTypePools.Init(replyType)

	return
}
