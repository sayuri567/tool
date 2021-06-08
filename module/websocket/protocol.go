package websocket

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

const ErrorMessageType uint32 = ^uint32(0)

type Context struct {
	wsMegType int
	conn      *Conn
	requestId string
	msgType   uint32
	isEnd     uint16

	Error *Error
	Data  proto.Message
}

func (this *Context) SendClient(Data proto.Message) {
	this.send(Data, nil, 0)
}

func (this *Context) SendError(err *Error) {
	this.send(nil, err, 0)
}

func (this *Context) send(data proto.Message, err *Error, isEnd uint16) {
	this.conn.Write(&Context{wsMegType: this.wsMegType, requestId: this.requestId, msgType: this.msgType, isEnd: isEnd, Data: data, Error: err})
}

type Error struct {
	Code    uint32
	Message string
}

func (this *Error) String() string {
	return fmt.Sprintf("code: %v, message: %v", this.Code, this.Message)
}

func NewError(code uint32, msg string) *Error {
	return &Error{Code: code, Message: msg}
}
