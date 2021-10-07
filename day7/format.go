package day7

import (
	"reflect"
	"time"
)

//format包里面规范了传输报文的格式，比如option和header

type Option struct {
	MagicNum int
	Codec    codecType //规定了报文解码的方式

	//规定了TCP连接和发送option的时间
	connectTime  time.Duration

	//规定了服务器call和send的时间
	handleTime  time.Duration

}
var DefaultOpt = &Option{
	MagicNum: 0x1129,
	Codec:    gobType,

	connectTime: 15*time.Second,
	handleTime: 10*time.Second,
}

type Header struct {
	ServerName string
	Err        string
	Seq        uint64
}
//形成的request需要包含请求的对象名和方法名，还有参数,reply的意义是啥暂时母鸡
type Request struct {
	head  *Header
	argv reflect.Value
	reply reflect.Value
	obj  *Object
	method *Method
}
type ClientResult struct {
	client *Client
	err error
}

