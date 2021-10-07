
package day5

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type gobCodec struct {
	con io.ReadWriteCloser
	dec *gob.Decoder
	enc *gob.Encoder

	buf *bufio.Writer
}

func NewGobCodec(con io.ReadWriteCloser)codeC{
	codec:= &gobCodec{
		con: con,
		dec: gob.NewDecoder(con),
		enc: nil,
		buf: nil,
	}
	codec.buf=bufio.NewWriter(con)
	codec.enc=gob.NewEncoder(codec.buf)
	return codec
}



func (g *gobCodec) Close() error {
	return  g.con.Close()
}

func (g *gobCodec) ReadHead(header *Header) error {
	return g.dec.Decode(header)
}

func (g *gobCodec) ReadBody(body interface{}) error {
	return  g.dec.Decode(body)
}

func (g *gobCodec) Write(header *Header, body interface{})(err error) {
	defer func() {
		g.buf.Flush()
		if err!=nil{
			_ = g.Close()
		}
	}()
	err = g.enc.Encode(header)
	if err!=nil{
		log.Println("gob header encoder error:",err)
		return
	}
	err = g.enc.Encode(body)
	if err!=nil{
		log.Println("gob body encoder error",err)
		return
	}
	return nil
}

