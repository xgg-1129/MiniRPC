package day4

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
)

type Server struct {
	MagicNum int
	Services sync.Map

	handTimeOut time.Duration
}

var DefaultServer =NewServer()
func NewServer()*Server  {
	return &Server{MagicNum:0x1129}
}
func (s *Server) Run(addr string)error  {
	listen, err := net.Listen("tcp", addr)
	if err!=nil{
		log.Println("Server listen tcp error",err)
		return err
	}
	for {
		con, err1 := listen.Accept()
		if err1!=nil{
			log.Println("Server accept tcp error")
			return err
		}
		go s.ServerCon(con)
	}
}

func (s *Server) ServerCon(con io.ReadWriteCloser) {
	defer con.Close()
	//首先检查option
	var opt Option
	if err := json.NewDecoder(con).Decode(&opt);err!=nil{
		log.Println("Server decode option err:",err)
		return
	}
	if opt.MagicNum!=s.MagicNum{
		log.Println("MagicNum is unable to match:")
		return
	}
	s.handTimeOut=opt.handleTime
	codefun:=newCodecFunMap[opt.Codec]
	if codefun==nil{
		log.Printf("can not find the codecFun of %s",opt.Codec)
		log.Println("------------------------------------")
		return
	}
	
	s.serverCodec(codefun(con))
}

func (s *Server) serverCodec(c codeC) {
	sendMutex:=new(sync.Mutex)
	wg:=new(sync.WaitGroup)

	for{
		req,err:=s.ReadQuqest(c)
		if err!=nil{
			//这里其实很傻逼，因为readQuest函数只会返req和nil，或者nil和err，不存在返回2个nil的情况
			if req == nil{
				break
			}
			req.head.Err=err.Error()
			s.sendResponse(c,sendMutex,req.head,struct{}{})
			continue
		}
		wg.Add(1)
		go s.serverHandleRequest(c,req,sendMutex,wg)
	}
	wg.Wait()
}

func (s *Server) ReadQuqest(c codeC)(*Request,error){
	head,err := s.ReadQuestHead(c)
	if err!=nil{
		return nil,err
	}
	req:=&Request{}
	req.head=head
	req.obj,req.method,err = s.FindService(head.ServerName)
	if err!=nil{
		return nil,err
	}
	//初始化为默认值
	req.reply=req.method.newReply()
	req.argv=req.method.newArgv()


	//给参数复制，因为arv是值类型的，所以要获取一个指向它的接口
	pointRegv:=req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr{
		pointRegv=req.argv.Addr().Interface()
	}
	if err = c.ReadBody(pointRegv); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}

	return req,nil
}

func (s *Server) ReadQuestHead(c codeC)(*Header,error){
	var head Header
	if err := c.ReadHead(&head);err!=nil{
		if err != io.EOF{
			log.Println("Server ReadHeader error",err)
		}
		return nil,err
	}
	return &head,nil
}

func (s *Server) sendResponse(cc codeC,send *sync.Mutex,header *Header,body interface{}) {
	send.Lock()
	defer send.Unlock()
	if err := cc.Write(header, body);err!=nil{
		log.Println("Server send header and body err:",err)
	}
}

func (s *Server) serverHandleRequest(cc codeC,req *Request,send *sync.Mutex,wg *sync.WaitGroup) {
	defer wg.Done()
	called:=make(chan struct{})
	sent:=make(chan struct{})
	go func() {
		err := req.obj.call(req.method, req.argv, req.reply)
		called<- struct{}{}
		if err!=nil{
			req.head.Err = err.Error()
			s.sendResponse(cc, send, req.head, struct {}{})
			return
		}
		s.sendResponse(cc, send, req.head,req.reply.Interface())
		sent<- struct{}{}
	}()
	if s.handTimeOut==0{
		<-called
		<-sent
		return
	}
	select{
	case <-time.After(s.handTimeOut):
		req.head.Err=fmt.Sprintf("Server call service time out")
		s.sendResponse(cc,send,req.head, struct {}{})
	case <-called:
		<-sent
	}



}
func Register(rcvr interface{}) error { return DefaultServer.RegisterObject(rcvr) }





// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServerCon(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }