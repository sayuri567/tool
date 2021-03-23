package model

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
	gorp "gopkg.in/gorp.v1"
)

type MysqlManager struct {
	*module.DefaultModule
	modelMap      map[string][]modelMapItem
	callbacks     []func()
	enableDbTrace bool
	inited        bool
	connStrGetter dbConnectionStringGetter
}

// Model Model.
type Model interface {
	SetDbMap(dbMap *gorp.DbMap)
	DbMap() *gorp.DbMap
	SetDb(db *sql.DB)
	Db() *sql.DB
	// 初始化结构体与数据表的绑定，如有需要，可自行实现
	Initer(dbMap *gorp.DbMap, obj interface{}, tableName string) error
	// 获取当前model绑定的表名
	GetTable() string
	SetModel(model Model)
	SetFields(fields string)
}

type modelMapItem struct {
	model Model
	obj   interface{}
}

// DbConnectionStringGetter DbConnectionStringGetter.
type dbConnectionStringGetter interface {
	GetDbConnectionString(dbKey string) string
}

// 单例
var mysqlManager = &MysqlManager{
	modelMap:  make(map[string][]modelMapItem),
	callbacks: make([]func(), 0),
	inited:    false,
}

func GetMysqlManager() *MysqlManager {
	return mysqlManager
}

func SetConnStrGetter(getter dbConnectionStringGetter) {
	mysqlManager.connStrGetter = getter
}

func (this *MysqlManager) Init() error {
	if this.connStrGetter == nil {
		return errors.New("connStrGetter not set")
	}
	for dbKey := range this.modelMap {
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
		mapItems, ok := this.modelMap[dbKey]
		if ok {
			for _, mi := range mapItems {
				mi.model.SetDbMap(dbMap)
				mi.model.SetDb(db)
				err = mi.model.Initer(dbMap, mi.obj, mi.model.GetTable())
				if err != nil {
					return err
				}
				mi.model.SetModel(mi.model)
				mi.model.SetFields(GetAllFieldsAsString(mi.obj))
			}
		}
	}
	this.modelMap = nil

	this.inited = true
	for _, callback := range this.callbacks {
		callback()
	}

	return nil
}

func (this *MysqlManager) Stop() {
	logrus.Info("Stopping mysql connects")
	for dbKey := range this.modelMap {
		mapItems, ok := this.modelMap[dbKey]
		if ok {
			for _, mi := range mapItems {
				err := mi.model.Db().Close()
				if err != nil {
					logrus.Error(err.Error())
				}
			}
		}
	}
	logrus.Info("Stopped mysql connects")
}

// Register Register.
func Register(dbKey string, model Model, obj interface{}) {
	mapItems, ok := mysqlManager.modelMap[dbKey]
	if ok {
		for _, mi := range mapItems {
			if model == mi.model {
				return
			}
			mysqlManager.modelMap[dbKey] = append(mapItems, modelMapItem{model: model, obj: obj})
		}
	} else {
		mapItems := make([]modelMapItem, 0, 5)
		mysqlManager.modelMap[dbKey] = append(mapItems, modelMapItem{model: model, obj: obj})
	}
}

// RegisterCallback RegisterCallback.
func RegisterCallback(callback func()) {
	if mysqlManager.inited {
		callback()
		return
	}
	mysqlManager.callbacks = append(mysqlManager.callbacks, callback)
}
