package websocket

import (
	"bytes"
	"encoding/binary"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sayuri567/gorun"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type Conn struct {
	sessionId string
	Data      map[string]interface{}

	conn      *websocket.Conn
	writeChan chan *Message
	closed    bool
	done      chan struct{}
	wg        sync.WaitGroup
	once      sync.Once
}

func (this *Conn) Write(msg *Message) {
	this.writeChan <- msg
}

func (this *Conn) GetSessionId() string {
	return this.sessionId
}

func (this *Conn) GetData(key string) interface{} {
	return this.Data[key]
}

func (this *Conn) SetData(key string, data interface{}) {
	this.Data[key] = data
}

func (this *Conn) serveIO() {
	this.done = make(chan struct{})
	this.writeChan = make(chan *Message, 100)
	this.wg.Add(2)
	go func() {
		this.write()
		this.wg.Done()
	}()
	go func() {
		this.read()
		this.wg.Done()
	}()
}

func (this *Conn) wait() {
	this.wg.Wait()
}

func (this *Conn) close(err error) {
	if this.closed {
		return
	}
	if err != nil {
		logrus.WithError(err).Error("close websocket connection for error")
	}
	this.once.Do(func() {
		this.conn.Close()
		close(this.done)
		delConn(this.sessionId)
		this.closed = true
	})
}

func (this *Conn) read() {
	for {
		tp, msg, err := this.conn.ReadMessage()
		if err != nil {
			this.close(err)
			break
		}

		var message = new(Message)
		message.wsMegType = tp
		message.MsgType = binary.LittleEndian.Uint32(msg[:4])
		message.ReplyType = binary.LittleEndian.Uint32(msg[4:8])
		if websocketModule.wsHandler.handlers[message.MsgType] != nil {
			protoref := websocketModule.wsHandler.handlers[message.MsgType].prototype
			if len(msg[8:]) > 0 && protoref != nil {
				err = proto.Unmarshal(msg[8:], protoref)
				if err != nil {
					logrus.WithError(err).Error("failed to decode message")
					continue
				}
			}
			message.Data = protoref
			gorun.Go(func(message *Message) {
				reply, err := websocketModule.wsHandler.handlers[message.MsgType].handler(this, message.Data)
				if reply != nil || err != nil {
					replyType := message.MsgType
					if message.ReplyType > 0 {
						replyType = message.ReplyType
					}
					this.Write(&Message{wsMegType: message.wsMegType, MsgType: replyType, Error: nil, Data: reply})
				}
			}, message)
		}
	}
}

func (this *Conn) write() {
	ticker := time.NewTicker(time.Second * 30)
loop:
	for {
		select {
		case msg := <-this.writeChan:
			if msg == nil {
				break loop
			}
			data, err := proto.Marshal(msg.Data)
			if err != nil {
				logrus.WithError(err).Error("failed to encode message")
			}

			header := make([]byte, 8)
			binary.LittleEndian.PutUint32(header[:4], msg.MsgType)
			binary.LittleEndian.PutUint32(header[4:], msg.ReplyType)
			err = this.conn.WriteMessage(msg.wsMegType, bytes.Join([][]byte{header, data}, []byte{}))
			if err != nil {
				logrus.WithError(err).Error("failed to write websocket message")
				break loop
			}
		case <-ticker.C:
			err := this.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				logrus.WithError(err).Error("failed to ping ws client")
				break loop
			}
		case <-this.done:
			break loop
		}
	}
	ticker.Stop()
	this.close(nil)
}
