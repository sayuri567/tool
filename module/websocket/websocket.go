package websocket

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sayuri567/tool/module"
	"github.com/sayuri567/tool/module/api"
	"github.com/sirupsen/logrus"
)

type WebsocketModule struct {
	*module.DefaultModule

	config    *Config
	conns     map[string]*Conn
	wsHandler *WsHandler
	rwLock    sync.RWMutex
}

type Config struct {
	// 如果apimodule有值，则使用现有apiModule，否则port必填，并且开启一个apiModule
	Port           int
	ApiModule      *api.ApiModule
	Path           string
	Interceptors   []func(*Conn) error
	MessageDecoder func([]byte) (interface{}, error)
	MessageEncoder func(interface{}) ([]byte, error)
}

var websocketModule = &WebsocketModule{
	conns:  make(map[string]*Conn),
	rwLock: sync.RWMutex{},
	wsHandler: &WsHandler{
		handlers: make(map[uint32]*handler),
	},
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 5 * time.Second,
	// 取消ws跨域校验
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GetWebsocketModule() *WebsocketModule {
	return websocketModule
}

func SetConfig(config *Config) {
	if config == nil {
		config = &Config{}
	}
	path := "/ws"
	if len(config.Path) > 0 {
		path = config.Path
	}
	websocketModule.config = config
	if config.ApiModule != nil {
		config.ApiModule.RegisterHandler(http.MethodGet, path, websocketModule.wsHandler.handlerHttp, nil)
	}
}

func GetConn(sessionId string) *Conn {
	websocketModule.rwLock.RLock()
	defer websocketModule.rwLock.RUnlock()
	return websocketModule.conns[sessionId]
}

func Broadcast(sessionIds []string, data *Message) {
	for _, sessionId := range sessionIds {
		conn := GetConn(sessionId)
		if conn == nil {
			continue
		}
		conn.Write(data)
	}
}

// TODO
func BroadcastAll(data *Message) {
	for _, conn := range websocketModule.conns {
		conn.Write(data)
	}
}

func setConn(sessionId string, conn *Conn) {
	websocketModule.rwLock.Lock()
	defer websocketModule.rwLock.Unlock()
	websocketModule.conns[sessionId] = conn
}

func delConn(sessionId string) {
	websocketModule.rwLock.Lock()
	defer websocketModule.rwLock.Unlock()
	delete(websocketModule.conns, sessionId)
}

func (this *WebsocketModule) Init() error {
	if this.config.ApiModule == nil && this.config.Port == 0 {
		return errors.New("ApiModule and Port must have one")
	}
	if len(this.config.Path) == 0 {
		this.config.Path = "/ws"
	}
	if this.config.ApiModule != nil {
		return nil
	}

	api.RegisterHandler(http.MethodGet, this.config.Path, websocketModule.wsHandler.handlerHttp, nil)
	api.GetApiModule().SetConfig(&api.Config{Address: fmt.Sprintf(":%v", this.config.Port), Mode: gin.ReleaseMode})
	logrus.Info("ws module inited")
	return api.GetApiModule().Init()
}

func (this *WebsocketModule) Run() error {
	if this.config.ApiModule == nil {
		return api.GetApiModule().Run()
	}
	logrus.Info("ws module started")
	return nil
}

func (this *WebsocketModule) Stop() {
	if (this.config.ApiModule) == nil {
		api.GetApiModule().Stop()
	}
	logrus.Info("ws module stopped")
}
