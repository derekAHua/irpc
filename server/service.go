package server

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unicode"
	"unicode/utf8"

	rerrors "github.com/derekAHua/irpc/errors"
	"github.com/derekAHua/irpc/log"
)

// Precompute the reflectType for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// Precompute the reflectType for context.
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
}

type functionType struct {
	sync.Mutex // protects counters
	fn         reflect.Value
	ArgType    reflect.Type
	ReplyType  reflect.Type
}

type service struct {
	name     string                   // name of service
	receiver reflect.Value            // receiver of methods for the service
	typ      reflect.Type             // type of the receiver
	method   map[string]*methodType   // registered methods
	function map[string]*functionType // registered functions
}

func isExported(name string) bool {
	runeName, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(runeName)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mType := method.Type
		mName := method.Name
		// Method must be exported
		if method.PkgPath != "" {
			continue
		}
		// Method needs four ins: receiver, context.Context, *args, *reply
		if mType.NumIn() != 4 {
			if reportErr {
				log.Debug("method ", mName, " has wrong number of ins:", mType.NumIn())
			}
			continue
		}
		// First arg must be context.Context
		ctxType := mType.In(1)
		if !ctxType.Implements(typeOfContext) {
			if reportErr {
				log.Debug("method ", mName, " must use context.Context as the first parameter")
			}
			continue
		}

		// Second arg need not be a pointer.
		argType := mType.In(2)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Info(mName, " parameter type not exported: ", argType)
			}
			continue
		}
		// Third arg must be a pointer.
		replyType := mType.In(3)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Info("method", mName, " reply type not a pointer:", replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Info("method", mName, " reply type not exported:", replyType)
			}
			continue
		}
		// Method needs one out.
		if mType.NumOut() != 1 {
			if reportErr {
				log.Info("method", mName, " has wrong number of outs:", mType.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mType.Out(0); returnType != typeOfError {
			if reportErr {
				log.Info("method", mName, " returns ", returnType.String(), " not error")
			}
			continue
		}
		methods[mName] = &methodType{method: method, ArgType: argType, ReplyType: replyType}

		// init pool for reflect.Type of args and reply
		reflectTypePools.Init(argType)
		reflectTypePools.Init(replyType)
	}
	return methods
}

// UnregisterAll unregisters all services.
// You can call this method when you want to shutdown/upgrade this node.
func (s *Server) UnregisterAll() error {
	s.serviceMapMu.RLock()
	defer s.serviceMapMu.RUnlock()
	var es []error
	for k := range s.serviceMap {
		err := s.Plugins.DoUnregister(k)
		if err != nil {
			es = append(es, err)
		}
	}

	if len(es) > 0 {
		return rerrors.NewMultiError(es)
	}
	return nil
}

func (s *service) call(ctx context.Context, mType *methodType, argv, reply reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			err = fmt.Errorf("[service internal error]: %v, method: %s, argv: %+v, stack: %s",
				r, mType.method.Name, argv.Interface(), buf)
			log.Error(err)
		}
	}()

	function := mType.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.receiver, reflect.ValueOf(ctx), argv, reply})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	if errInter != nil {
		return errInter.(error)
	}

	return
}

func (s *service) callForFunction(ctx context.Context, ft *functionType, argv, reply reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			// log.Errorf("failed to invoke service: %v, stacks: %s", r, string(debug.Stack()))
			err = fmt.Errorf("[service internal error]: %v, function: %s, argv: %+v, stack: %s",
				r, runtime.FuncForPC(ft.fn.Pointer()), argv.Interface(), buf)
			log.Error(err)
		}
	}()

	// Invoke the function, providing a new value for the reply.
	returnValues := ft.fn.Call([]reflect.Value{reflect.ValueOf(ctx), argv, reply})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	if errInter != nil {
		return errInter.(error)
	}

	return
}
