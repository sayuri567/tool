package websocket

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	writeChan chan *Context
	closed    bool
	done      chan struct{}
	wg        sync.WaitGroup
	once      sync.Once
}

func (this *Conn) Write(msg *Context) {
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
	this.writeChan = make(chan *Context, 100)
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

// header: 0-3:请求类型，4-5:是否有错误，6-7:是否结束本次请求，8-43:请求id
func (this *Conn) read() {
	for {
		tp, msg, err := this.conn.ReadMessage()
		if err != nil {
			this.close(err)
			break
		}

		ctx, err := this.unmarshal(msg)
		if err != nil {
			logrus.WithError(err).Error("failed to decode message")
			continue
		}
		ctx.wsMegType = tp
		ctx.conn = this
		if websocketModule.wsHandler.handlers[ctx.msgType] != nil {
			gorun.Go(func(ctx *Context) {
				reply, err := websocketModule.wsHandler.handlers[ctx.msgType].handler(ctx, ctx.Data)
				ctx.send(reply, err, 1)
			}, ctx)
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
			buffer, err := this.marshal(msg)
			if err != nil {
				logrus.WithError(err).Error("failed to encode message")
				continue
			}
			err = this.conn.WriteMessage(msg.wsMegType, buffer)
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

func (this *Conn) marshal(ctx *Context) ([]byte, error) {
	var data []byte
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[:4], ctx.msgType) // 请求类型
	binary.LittleEndian.PutUint16(header[6:8], ctx.isEnd)  // 是否结束请求
	header = append(header, []byte(ctx.requestId)...)      // 请求id
	if ctx.Error == nil {
		binary.LittleEndian.PutUint16(header[4:6], 0) // 是否有错误
		buffer, err := proto.Marshal(ctx.Data)
		if err != nil {
			logrus.WithError(err).Error("failed to encode message")
			return nil, err
		}
		data = bytes.Join([][]byte{header, buffer}, []byte{})
	} else {
		binary.LittleEndian.PutUint16(header[4:6], 1) // 是否有错误
		errCode := make([]byte, 4)
		binary.LittleEndian.PutUint32(errCode[:4], ctx.Error.Code)
		data = bytes.Join([][]byte{header, errCode, []byte(ctx.Error.Message)}, []byte{})
	}

	return data, nil
}

func (this *Conn) unmarshal(msg []byte) (*Context, error) {
	var ctx = new(Context)
	if len(msg) < 44 {
		return ctx, errors.New("invalid request")
	}
	ctx.msgType = binary.LittleEndian.Uint32(msg[:4]) // 请求类型
	// hasError := binary.LittleEndian.Uint16(msg[4:6]) // 是否有错误
	// isEnd := binary.LittleEndian.Uint16(msg[6:8]) // 是否结束请求
	ctx.requestId = string(msg[8:44]) // 请求id
	if websocketModule.wsHandler.handlers[ctx.msgType] != nil && websocketModule.wsHandler.handlers[ctx.msgType].prototype != nil {
		protoref := websocketModule.wsHandler.handlers[ctx.msgType].prototype.ProtoReflect().New().Interface()
		err := proto.Unmarshal(msg[44:], protoref)
		if err != nil {
			return nil, err
		}
		ctx.Data = protoref
	}

	return ctx, nil
}
