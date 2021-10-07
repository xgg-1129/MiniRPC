package day7

import (
	"errors"
	"go/ast"
	"log"
	"reflect"
	"strings"
	"sync/atomic"
)

type Method struct {
	method   reflect.Method
	Argv     reflect.Type
	Reply    reflect.Type
	NumCalls uint64
}

func (m *Method) newArgv() reflect.Value {
	var argv reflect.Value
	// arg may be a pointer type, or a value type
	if m.Argv.Kind() == reflect.Ptr {
		argv = reflect.New(m.Argv.Elem())
	} else {
		//如果他不是指针，为啥还要加elem
		//reflect.New返回的是指针value
		argv = reflect.New(m.Argv).Elem()
	}
	return argv
}

func (m *Method) newReply() reflect.Value {
	replyv := reflect.New(m.Reply.Elem())
	switch m.Reply.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.Reply.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.Reply.Elem(), 0, 0))
	}
	return replyv
}

type Object struct {
	name string
	objType reflect.Type
	objValue reflect.Value
	methods map[string]*Method
}

func NewObject(object interface{}) *Object {
	//这里传递的是指针类型的object
	res:=new(Object)
	res.objType=reflect.TypeOf(object)
	res.objValue=reflect.ValueOf(object)
	//Indirect(v)会返回v指向的值，如果v不是指针，则返回他自己的值
	res.name=reflect.Indirect(res.objValue).Type().Name()
	if !ast.IsExported(res.name) {
		log.Fatalf("rpc server: %s is not a valid service name", res.name)
	}
	res.registerMethods()
	return res
}
func (obj *Object) registerMethods() {
	obj.methods = make(map[string]*Method)
	for i := 0; i < obj.objType.NumMethod(); i++ {
		method := obj.objType.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		obj.methods[method.Name] = &Method{
			method: method,
			Argv:   argType,
			Reply:  replyType,
		}
		log.Printf("rpc server: register %s.%s\n", obj.name, method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *Server) RegisterObject(object interface{}) error {
	obj:=NewObject(object)
	if _, ok := s.Services.LoadOrStore(obj.name, obj);ok{
		return errors.New("rpc: service already defined: " + obj.name)
	}
	return nil
}

func (s *Server) FindService(serverName string) (obj *Object,method *Method,err error){
	dot:=strings.LastIndex(serverName,".")
	if dot<0{
		err=errors.New("rpc server: service/method request ill-formed: " + serverName)
		return
	}
	serviceName, methodName := serverName[:dot],serverName[dot+1:]
	o, ok := s.Services.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	obj = o.(*Object)
	method = obj.methods[methodName]
	if method == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}
func (obj *Object) call(m *Method, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.NumCalls, 1)
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{obj.objValue, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}


