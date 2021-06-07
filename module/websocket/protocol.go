package websocket

import "google.golang.org/protobuf/proto"

type Message struct {
	wsMegType int
	MsgType   uint32
	ReplyType uint32
	Error     proto.Message
	Data      proto.Message
}
