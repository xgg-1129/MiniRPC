package day1

import (

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

type Server struct {
	MagicNum int
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
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = c.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
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
	log.Println(req.head, req.argv.Elem())
	req.reply = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.head.Seq))
	s.sendResponse(cc, send, req.head,req.reply.Interface())
}





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