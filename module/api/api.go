package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sayuri567/gorun"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// 监听的端口
	Address string

	// 为分组添加不同的过滤器
	// key为url的前缀，如key为/v1，那么就是为/v1为前缀的接口添加过滤器，验证等等
	GroupFilter map[string][]gin.HandlerFunc

	// 非必需，如果为空，则自动使用gin.New()
	Gin *gin.Engine

	// 全局的过滤器
	GlobalFilter []gin.HandlerFunc

	// GinMode
	Mode string

	AccessLog bool

	SaveOperation SaveOperation
}

type ApiModule struct {
	*module.DefaultModule

	config        *Config
	server        *http.Server
	handlers      map[string]*httpHandler
	sensitiveKeys []string
}

var apiModule = New()

func New() *ApiModule {
	return &ApiModule{
		handlers:      make(map[string]*httpHandler),
		sensitiveKeys: make([]string, 0),
	}
}

func GetApiModule() *ApiModule {
	return apiModule
}

// RegisterHandler ("Get", /angel/strength", game_angel_strentgh, &api.AngelStrengthParam{})
func (this *ApiModule) RegisterHandler(method string, path string,
	handler func(*gin.Context, interface{}) (interface{}, error),
	prototype interface{}) {

	this.handlers[method+":"+path] = &httpHandler{
		method:    method,
		handler:   handler,
		prototype: prototype,
		path:      path,
		module:    this,
	}
}

func RegisterHandler(method string, path string,
	handler func(*gin.Context, interface{}) (interface{}, error),
	prototype interface{}) {
	apiModule.RegisterHandler(method, path, handler, prototype)
}

// SetSensitiveKeys SetSensitiveKeys
func (this *ApiModule) SetSensitiveKeys(keys []string) {
	this.sensitiveKeys = append(this.sensitiveKeys, keys...)
}

// SetSensitiveKeys SetSensitiveKeys
func SetSensitiveKeys(keys []string) {
	apiModule.SetSensitiveKeys(keys)
}

func (this *ApiModule) SetConfig(config *Config) {
	this.config = config
}

func SetConfig(config *Config) {
	apiModule.SetConfig(config)
}

func (this *ApiModule) Init() error {
	if len(this.config.Address) == 0 {
		this.config.Address = ":8080"
	}
	if len(this.config.Mode) == 0 {
		this.config.Mode = gin.ReleaseMode
	}
	gin.SetMode(this.config.Mode)
	if this.config.Gin == nil {
		this.config.Gin = gin.New()
	}
	for _, filter := range this.config.GlobalFilter {
		this.config.Gin.Use(filter)
	}
	this.config.Gin.NoRoute(func(g *gin.Context) {
		g.JSON(http.StatusNotFound, Output{Code: 404, Message: NOT_FOUND})
	})

	for prefix, filters := range this.config.GroupFilter {
		group := this.config.Gin.Group(prefix)
		for _, filter := range filters {
			group.Use(filter)
		}
		for key, handler := range this.handlers {
			if !strings.HasPrefix(handler.path, prefix) {
				continue
			}
			group.Handle(handler.method, handler.path[len(prefix):], handler.ServeHTTP)
			delete(this.handlers, key)
		}
	}

	// such as : /login, /register
	for key, handler := range this.handlers {
		this.config.Gin.Handle(handler.method, handler.path, handler.ServeHTTP)
		delete(this.handlers, key)
	}

	this.server = &http.Server{Addr: this.config.Address, Handler: this.config.Gin}

	logrus.Info("gin module inited")
	return nil
}

func (this *ApiModule) Run() error {
	gorun.Go(func() {
		err := this.server.ListenAndServe()
		if err != nil {
			logrus.WithError(err).Error("failed to start gin")
		}
	})
	logrus.Infof("gin module listen %v", this.config.Address)
	return nil
}

func (this *ApiModule) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	this.server.Shutdown(ctx)
}
