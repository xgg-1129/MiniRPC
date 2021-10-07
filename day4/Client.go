package day4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Call struct {
	ServerName string
	Seq uint64

	Argv interface{}
	Reply interface{}
	Err error
	Done chan *Call
}

func (c *Call) done()  {
	c.Done <- c
}
type Client struct {
	mu sync.Mutex
	sendMutex sync.Mutex
	pending map[uint64]*Call
	opt Option

	index uint64
	cc codeC
}

func NewClient(con net.Conn,opt *Option)(*Client,error){
	funcode:=newCodecFunMap[opt.Codec]
	if funcode == nil{
		err:=fmt.Errorf("invaild codec type %s",opt.Codec)
		return nil,err
	}
	if err := json.NewEncoder(con).Encode(opt);err!=nil{
		return nil,err
	}
	return NewClientCodec(funcode(con)),nil
}

func NewClientCodec(conn codeC) *Client {
	client:=new(Client)
	client.cc=conn
	client.index=1
	client.pending=make(map[uint64]*Call)
	go client.reveive()
	return client
}
func Dial(addr string,opt *Option)(client *Client,err error){

	con, err := net.DialTimeout("tcp", addr,opt.connectTime)
	defer func() {
		if client==nil{
			//关闭socket是关闭连接，还是关闭socket文件呢
			con.Close()
		}
	}()
	if err!=nil{
		log.Println("dial conn error :",err)
		return nil,err
	}

	ch := make(chan ClientResult)
	go func() {
		client,err=NewClient(con,opt)
		//ch是属于谁的资源?
		ch<-ClientResult{
			client: client,
			err:    err,
		}
	}()
	if opt.connectTime==0{
		res:=<-ch
		return res.client,nil
	}
	select {
	case res:=<-ch:
		return res.client,res.err
	case <-time.After(opt.connectTime):
		return nil,errors.New("send option time out")
	}
}
func (c *Client) registerCall(call *Call)(seq uint64,err error)  {
	c.mu.Lock()
	defer c.mu.Unlock()

	call.Seq=c.index
	c.index++
	c.pending[call.Seq]=call
	return call.Seq,nil
}
func (c *Client) RemoveCall(seq uint64) *Call{
	c.mu.Lock()
	defer c.mu.Unlock()
	res:=c.pending[seq]
	delete(c.pending,seq)
	return res
}
//由于某种原因，需要删除掉所有待定的call
func (c *Client) terminateCalls(err error)  {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _,item := range c.pending{
		item.Err=err
		item.done()
	}
}

func (c *Client) reveive()  {
	var err error
	for{
		var head Header
		if err = c.cc.ReadHead(&head);err!=nil{
			break
		}
		call := c.RemoveCall(head.Seq)
		switch {
		case call==nil:
			err=c.cc.ReadBody(nil)
		case head.Err!="":
			err=errors.New(head.Err)
			call.done()
		default:
			err=c.cc.ReadBody(call.Reply)
			call.Err=err
			call.done()
		}
		if err!=nil{
			log.Println("receive error:",err)
		}
	}
	c.terminateCalls(err)
}
func (c *Client) send(call *Call)  {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	seq, err := c.registerCall(call)
	if err!=nil{
		call.Err=err
		call.done()
		return
	}
	var head Header
	head.Seq=seq
	head.ServerName=call.ServerName

	if err = c.cc.Write(&head, call.Argv);err!=nil{
		c.RemoveCall(seq)
		call.Err=err
		call.done()
		return
	}
}

//不是很懂这个done参数的意义，done能控制异步的数量
func (c *Client) AsyCall(ServerName string, argv, reply interface{}, done chan *Call) *Call {
	if done == nil{
		done = make(chan*Call,10)
	}else if cap(done)==0{
		log.Panic("client:AsyCall  done is unbuffered")
	}
	call:=&Call{
		ServerName: ServerName,
		Argv:       argv,
		Reply:      reply,
		Done:       done,
	}
	go c.send(call)
	return call
}

func (c *Client) SynCall(ServerName string, argv, reply interface{}) error {
	call:=<-c.AsyCall(ServerName,argv,reply,make(chan*Call,1)).Done
	return call.Err
}
func (c *Client) TimeOutSynCall(ctx context.Context,ServerName string, argv, reply interface{})error{
	call := c.AsyCall(ServerName, argv, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		removeCall := c.RemoveCall(call.Seq)
		removeCall.Err=errors.New("reveive time out")
		return removeCall.Err
	case res:=<-call.Done:
		return res.Err
	}
}
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.cc.Close()
}