package day5

import (
	"io"

)


type codeC interface {
	io.Closer
	ReadHead(*Header)error
	ReadBody(interface{})error
	Write(*Header,interface{})error
}
type newCodecFun func(conn io.ReadWriteCloser)codeC
//如果这里用的是
type codecType string

const (
	gobType  codecType = "gob"
)

var newCodecFunMap map[codecType]newCodecFun

func init() {
	newCodecFunMap=make(map[codecType]newCodecFun)
	newCodecFunMap[gobType]=NewGobCodec
}
