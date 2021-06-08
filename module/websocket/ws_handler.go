package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type WsHandler struct {
	handlers map[uint32]*handler
}

type handler struct {
	msgType   uint32
	handler   func(*Context, proto.Message) (proto.Message, *Error)
	prototype proto.Message
}

func RegisterHandler(msgType uint32, msgHandler func(*Context, proto.Message) (proto.Message, *Error), prototype proto.Message) {
	websocketModule.wsHandler.handlers[msgType] = &handler{
		msgType:   msgType,
		handler:   msgHandler,
		prototype: prototype,
	}
}

func (this *WsHandler) handlerHttp(g *gin.Context, p interface{}) (interface{}, error) {
	var err error
	var conn *websocket.Conn
	conn, err = wsupgrader.Upgrade(g.Writer, g.Request, nil)
	if err != nil {
		return nil, err
	}
	wsConn, err := this.interceptor(conn)
	if err != nil {
		return nil, err
	}

	setConn(wsConn.sessionId, wsConn)

	wsConn.serveIO()
	wsConn.wait()
	g.Abort()

	return nil, nil
}

func (this *WsHandler) interceptor(conn *websocket.Conn) (*Conn, error) {
	var err error

	wsconn := &Conn{conn: conn, sessionId: uuid.NewV4().String()}
	if len(websocketModule.config.Interceptors) > 0 {
		for _, interceptor := range websocketModule.config.Interceptors {
			err = interceptor(wsconn)
			if err != nil {
				if closeErr := wsconn.conn.Close(); closeErr != nil {
					logrus.WithError(closeErr).Error("failed to close wsconn")
				}
			}
		}
	}
	return wsconn, err
}
