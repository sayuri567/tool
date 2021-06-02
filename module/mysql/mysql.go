package mysql

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	"github.com/sayuri567/tool/base/model"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
	gorp "gopkg.in/gorp.v1"
)

type MysqlModule struct {
	*module.DefaultModule
	modelMap      map[string][]modelMapItem
	callbacks     []func()
	enableDbTrace bool
	inited        bool
	connStrGetter dbConnectionStringGetter
}

type modelMapItem struct {
	model model.Model
	obj   interface{}
}

// DbConnectionStringGetter DbConnectionStringGetter.
type dbConnectionStringGetter interface {
	GetDbConnectionString(dbKey string) string
}

// 单例
var mysqlModule = &MysqlModule{
	modelMap:  make(map[string][]modelMapItem),
	callbacks: make([]func(), 0),
	inited:    false,
}

func GetMysqlModule() *MysqlModule {
	return mysqlModule
}

func SetConnStrGetter(getter dbConnectionStringGetter) {
	mysqlModule.connStrGetter = getter
}

func (this *MysqlModule) Init() error {
	if this.connStrGetter == nil {
		return errors.New("connStrGetter not set")
	}
	for dbKey, mapItems := range this.modelMap {
		db, err := sql.Open("mysql", this.connStrGetter.GetDbConnectionString(dbKey))
		if err != nil {
			return err
		}
		err = db.Ping()
		if err != nil {
			return err
		}
		db.SetMaxIdleConns(10)
		db.SetMaxOpenConns(100)
		db.SetConnMaxLifetime(200 * time.Second)
		dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{}}
		if this.enableDbTrace {
			dbMap.TraceOn("", &dbLogger{})
		}
		for _, mi := range mapItems {
			mi.model.SetDbMap(dbMap)
			mi.model.SetDb(db)
			err = mi.model.Initer(dbMap, mi.obj, mi.model.GetTable())
			if err != nil {
				return err
			}
			mi.model.SetModel(mi.model)
			mi.model.SetFields(model.GetAllFieldsAsString(mi.obj))
		}
	}

	this.inited = true
	for _, callback := range this.callbacks {
		callback()
	}

	logrus.Info("mysql module inited")
	return nil
}

func (this *MysqlModule) Stop() {
	logrus.Info("Stopping mysql connects")
	for _, mapItems := range this.modelMap {
		for _, mi := range mapItems {
			err := mi.model.Db().Close()
			if err != nil {
				logrus.Error(err.Error())
			}
			break
		}
	}
	logrus.Info("Stopped mysql connects")
}

// Register Register.
func Register(dbKey string, model model.Model, obj interface{}) {
	mapItems, ok := mysqlModule.modelMap[dbKey]
	if ok {
		for _, mi := range mapItems {
			if model == mi.model {
				return
			}
			mysqlModule.modelMap[dbKey] = append(mapItems, modelMapItem{model: model, obj: obj})
		}
	} else {
		mapItems := make([]modelMapItem, 0, 5)
		mysqlModule.modelMap[dbKey] = append(mapItems, modelMapItem{model: model, obj: obj})
	}
}

// RegisterCallback RegisterCallback.
func RegisterCallback(callback func()) {
	if mysqlModule.inited {
		callback()
		return
	}
	mysqlModule.callbacks = append(mysqlModule.callbacks, callback)
}

// DbLogger DbLogger.
type dbLogger struct {
}

// Printf Printf.
func (this *dbLogger) Printf(format string, v ...interface{}) {
	logrus.WithFields(logrus.Fields{"@type": "mysql"}).Infof(format, v...)
}
