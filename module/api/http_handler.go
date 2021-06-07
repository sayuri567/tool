package api

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type httpHandler struct {
	method    string
	path      string
	handler   func(*gin.Context, interface{}) (interface{}, error)
	prototype interface{} //接口参数
	module    *ApiModule
}

type SaveOperation interface {
	Save(method, uri, ip string, params, userData map[string]interface{})
}

// ServeHTTP
func (this *httpHandler) ServeHTTP(c *gin.Context) {
	var logDatas map[string]interface{}
	var err error
	var param interface{}
	if this.prototype != nil {
		param = reflect.New(reflect.TypeOf(this.prototype).Elem()).Interface()
		err = c.Bind(param)
		if this.module.config.AccessLog || this.module.config.SaveOperation != nil {
			logDatas = this.getData(param)
		}
	}

	if this.module.config.AccessLog {
		logrus.WithFields(logrus.Fields{"uri": c.Request.RequestURI, "@type": "access", "params": logDatas, "method": c.Request.Method, "ip": c.ClientIP()}).Info("access log")
	}
	if this.module.config.SaveOperation != nil {
		this.module.config.SaveOperation.Save(c.Request.Method, c.Request.RequestURI, c.ClientIP(), logDatas, c.Keys)
	}
	if err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, &Output{Code: 0, Message: err.Error(), Data: ""})
		return
	}

	data, err := this.handler(c, param)
	if c.IsAborted() {
		return
	}
	output := &Output{Code: 1, Message: SUCCESS, Data: data}
	if err != nil {
		output.Message = err.Error()
		output.Code = 0
		c.JSON(http.StatusBadRequest, output)
		return
	}
	c.JSON(http.StatusOK, output)
}

func (this *httpHandler) getData(param interface{}) map[string]interface{} {
	// 敏感字段屏蔽，待优化
	jsonDatas, _ := json.Marshal(param)
	logDatas := map[string]interface{}{}
	json.Unmarshal(jsonDatas, &logDatas)
	for _, key := range this.module.sensitiveKeys {
		if _, ok := logDatas[key]; ok {
			logDatas[key] = "***"
		}
	}

	return logDatas
}
