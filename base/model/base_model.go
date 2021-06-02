package model

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	gorp "gopkg.in/gorp.v1"
)

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

// CommonModel CommonModel.
type CommonModel struct {
	dbMap *gorp.DbMap
	db    *sql.DB

	fields string
	model  Model
}

// BaseModel BaseModel.
type BaseModel struct {
	SimpleModel
	Id          int       `db:"id" json:"id"`
	CreatedTime time.Time `db:"createdTime" json:"createdTime"`
	IsDeleted   int       `db:"isDeleted" json:"isDeleted"`
	UpdatedTime time.Time `db:"updatedTime" json:"updatedTime"`
}

type LogModel struct {
	SimpleModel
	Id          int       `db:"id" json:"id"`
	CreatedTime time.Time `db:"createdTime" json:"createdTime"`
}

type SimpleModel struct {
}

// QueryItem QueryItem.
type QueryItem []interface{}

// QueryMap QueryMap.
type QueryMap map[string]interface{}

// SetDbMap SetDbMap.
func (this *CommonModel) SetDbMap(dbMap *gorp.DbMap) {
	this.dbMap = dbMap
}

// DbMap DbMap.
func (this *CommonModel) DbMap() *gorp.DbMap {
	return this.dbMap
}

// SetDb SetDb.
func (this *CommonModel) SetDb(db *sql.DB) {
	this.db = db
}

// Db Db.
func (this *CommonModel) Db() *sql.DB {
	return this.db
}

func (this *CommonModel) GetTable() string {
	panic("Func GetTable must be implemented")
}

func (this *CommonModel) SetModel(model Model) {
	this.model = model
}

func (this *CommonModel) GetModel() Model {
	return this.model
}

func (this *CommonModel) SetFields(fields string) {
	this.fields = fields
}

func (this *CommonModel) GetFields() string {
	return this.fields
}

func (this *CommonModel) Initer(dbMap *gorp.DbMap, obj interface{}, tableName string) error {
	dbMap.AddTableWithName(obj, tableName).SetKeys(true, "id")
	return nil
}

// Create Create.
func (this *CommonModel) Create(model interface{}) (int, error) {
	m := reflect.ValueOf(model).Elem()
	err := this.DbMap().Insert(model)
	if err != nil {
		return 0, err
	}

	return m.FieldByName("Id").Interface().(int), nil
}

// UpdateById UpdateById.
func (this *CommonModel) UpdateById(fields map[string]interface{}, id ...int) (int, error) {
	if _, ok := fields["updatedTime"]; !ok {
		fields["updatedTime"] = time.Now()
	}

	ids := []string{}
	for _, item := range id {
		ids = append(ids, strconv.Itoa(item))
	}
	update := ""
	params := []interface{}{}
	for field, value := range fields {
		coma := ""
		if update != "" {
			coma = ","
		}
		params = append(params, value)
		update += fmt.Sprintf("%v `%v`=?", coma, field)
	}
	sql := fmt.Sprintf("update %v set %v where `id` in (%v) and `isDeleted`=0", this.GetModel().GetTable(), update, strings.Join(ids, ","))
	result, err := this.DbMap().Exec(sql, params...)
	if err != nil {
		return 0, err
	}

	row, err := result.RowsAffected()
	return int(row), err
}

// DeleteById DeleteById.
func (this *CommonModel) DeleteById(id ...string) (int, error) {
	sql := fmt.Sprintf("update %v set isDeleted=%v where `id` in (%v) and `isDeleted`=0", this.GetModel().GetTable(), 1, strings.Join(id, ","))
	result, err := this.DbMap().Exec(sql)
	if err != nil {
		return 0, err
	}

	row, err := result.RowsAffected()
	return int(row), err
}

// GetInterfaceById 通过id获取记录.
func (this *CommonModel) GetInterfaceById(dataType interface{}, id ...int) ([]interface{}, error) {
	if len(id) == 0 {
		return nil, nil
	}
	idsStr := ""
	for _, i := range id {
		if idsStr != "" {
			idsStr += ","
		}
		idsStr += strconv.Itoa(i)
	}
	sql := fmt.Sprintf("select %s from %s where `id` in (%v) and `isDeleted`=0", this.GetFields(), this.GetModel().GetTable(), idsStr)
	data, err := this.DbMap().Select(dataType, sql)

	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return data, nil
}

// SearchInterface 通用查询方法.
func (this *CommonModel) SearchInterface(page int, pageSize int, sort string, query map[string]string, genCondition func(map[string]string) QueryMap, dataType interface{}) ([]interface{}, int, error) {
	var total = 0
	var offset int
	if page <= 1 {
		offset = 0
	} else {
		offset = (page - 1) * pageSize
	}
	orderBy := this.ParseOrder(sort)

	whereStr, params := GenWhere(genCondition(query))
	pageParams := append(params, offset, pageSize)
	sql := fmt.Sprintf("select %v from %v where `isDeleted` = 0 and %v order by %v limit ?,?;", this.GetFields(), this.GetModel().GetTable(), whereStr, orderBy)
	list, err := this.DbMap().Select(dataType, sql, pageParams...)
	if err != nil {
		return nil, total, err
	}
	sql = fmt.Sprintf("select count(id) from %s where `isDeleted`=0 and %v", this.GetModel().GetTable(), whereStr)
	err = this.DbMap().SelectOne(&total, sql, params...)
	if err != nil {
		return nil, total, err
	}
	return list, total, nil
}

func (this *CommonModel) SearchAllInterface(sort string, query map[string]string, genCondition func(map[string]string) QueryMap, dataType interface{}) ([]interface{}, error) {
	var whereStr = " 1 "
	var params = []interface{}{}
	if genCondition != nil {
		whereStr, params = GenWhere(genCondition(query))
	}
	orderBy := this.ParseOrder(sort)
	sql := fmt.Sprintf("select %v from %v where `isDeleted` = 0 and %v order by %v;", this.GetFields(), this.GetModel().GetTable(), whereStr, orderBy)
	list, err := this.DbMap().Select(dataType, sql, params...)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// UpdateByCondition UpdateByCondition.
func (this *CommonModel) UpdateByCondition(fields map[string]interface{}, conditions QueryMap) (int, error) {
	if _, ok := fields["updatedTime"]; !ok {
		fields["updatedTime"] = time.Now()
	}
	update := ""
	params := []interface{}{}
	for field, value := range fields {
		coma := ""
		if update != "" {
			coma = ","
		}
		params = append(params, value)
		update += fmt.Sprintf("%v `%v`=?", coma, field)
	}
	whereStr, ps := GenWhere(conditions)
	params = append(params, ps...)
	sql := fmt.Sprintf("update %v set %v where %v and `isDeleted`=0", this.GetModel().GetTable(), update, whereStr)
	result, err := this.DbMap().Exec(sql, params...)
	if err != nil {
		return 0, err
	}

	row, err := result.RowsAffected()
	return int(row), err
}

// ParseOrder 格式化排序.
func (this *CommonModel) ParseOrder(sort string) string {
	sorts := strings.Split(sort, ",")
	res := ""
	for _, s := range sorts {
		order := strings.Split(strings.Trim(s, " "), " ")
		if len(order) > 0 && strings.Contains(this.GetFields(), order[0]) {
			if res != "" {
				res += ","
			}
			res += order[0]
			if len(order) > 1 && strings.ToLower(order[1]) == "desc" {
				res += " desc"
			}
		}
	}

	if len(res) == 0 {
		res = "id desc"
	}

	return res
}

// PreInsert 插入前操作.
func (this *BaseModel) PreInsert(s gorp.SqlExecutor) error {
	this.IsDeleted = 0
	if this.CreatedTime.IsZero() {
		this.CreatedTime = time.Now()
	}
	if this.UpdatedTime.IsZero() {
		this.UpdatedTime = time.Now()
	}
	return nil
}

// PreUpdate 更新前操作.
func (this *BaseModel) PreUpdate(s gorp.SqlExecutor) error {
	this.UpdatedTime = time.Now()
	return nil
}

// PreInsert 插入前操作.
func (this *LogModel) PreInsert(s gorp.SqlExecutor) error {
	if this.CreatedTime.IsZero() {
		this.CreatedTime = time.Now()
	}
	return nil
}

// PreInsert 插入前操作.
func (this *SimpleModel) PreInsert(s gorp.SqlExecutor) error {
	return nil
}

// PreUpdate 更新前操作.
func (this *SimpleModel) PreUpdate(s gorp.SqlExecutor) error {
	return nil
}

// PostGet 查询后.
func (this *SimpleModel) PostGet(s gorp.SqlExecutor) error {
	return nil
}

// PostInsert 插入后.
func (this *SimpleModel) PostInsert(s gorp.SqlExecutor) error {
	return nil
}

// PostUpdate 更新后.
func (this *SimpleModel) PostUpdate(s gorp.SqlExecutor) error {
	return nil
}

// PreDelete 删除前.
func (this *SimpleModel) PreDelete(s gorp.SqlExecutor) error {
	return nil
}

// PostDelete 删除后.
func (this *SimpleModel) PostDelete(s gorp.SqlExecutor) error {
	return nil
}

// GenFieldsStrWithTable 生成携带表明的字段sql.
func (this *CommonModel) GenFieldsStrWithTable(asName string, withoutAs ...bool) string {
	fields := strings.Split(this.GetFields(), ",")
	fieldsStr := ""
	tableName := this.GetModel().GetTable()
	if asName != "" {
		tableName = asName
	}
	isWithoutAs := false
	if withoutAs != nil && withoutAs[0] {
		isWithoutAs = true
	}
	for _, field := range fields {
		fieldsStr += "`" + tableName + "`." + field
		if !isWithoutAs {
			fieldsStr += " as " + field
		}
		fieldsStr += ","
	}
	return strings.Trim(fieldsStr, ",")
}

// GetAllFieldsAsString 获取model中的所有字段，防止select * 返回model中未定义的字段.
func GetAllFieldsAsString(obj interface{}) string {
	objT := reflect.TypeOf(obj)
	var fields []string
	for i := 0; i < objT.NumField(); i++ {
		fieldT := objT.Field(i)
		tag := fieldT.Tag.Get("db")
		if tag == "-" {
			continue
		}
		if tag == "" && (fieldT.Type.Kind() == reflect.Struct || fieldT.Type.Kind() == reflect.Ptr) {
			fieldType := fieldT.Type
			if fieldT.Type.Kind() == reflect.Ptr {
				fieldType = fieldT.Type.Elem()
			}
			for j := 0; j < fieldType.NumField(); j++ {
				fieldT := fieldType.Field(j)
				tag := fieldT.Tag.Get("db")
				if tag == "" || tag == "-" {
					continue
				}
				fields = append(fields, "`"+tag+"`")
			}
			continue
		}
		if tag == "" {
			continue
		}
		fields = append(fields, "`"+tag+"`")
	}
	return strings.Join(fields, ",")
}

// GenWhere 组装where语句
// e.g.
// 		QueryMap{
//			"$or":QueryMap{
//				"test4": QueryItem{"=", 1},
//				"test6": QueryItem{"in", QueryItem{3, 5}},
//			},
//			"test":  QueryItem{"=", 1},
//			"test":  QueryItem{"like", 1},
//			"test2": QueryItem{"in", QueryItem{1, 2, 3}},
//			"test3": QueryItem{"between", 3, 4},
//		}
func GenWhere(whereMap QueryMap, args ...interface{}) (string, []interface{}) {
	connector := " and "
	if len(args) > 0 {
		connector = args[0].(string)
	}
	where := ""
	params := []interface{}{}
	for key, value := range whereMap {
		if where != "" {
			where += connector
		}
		if key == "$or" {
			w, p := GenWhere(value.(QueryMap), " or ")
			where += "(" + w + ")"
			params = append(params, p...)
			continue
		}

		v := value.(QueryItem)
		op := v[0].(string)
		switch v[0] {
		case "in":
			vals := v[1].(QueryItem)
			if vals == nil || len(vals) == 0 {
				continue
			}
			valsStr := ""
			for _, vv := range vals {
				valsStr += ",?"
				params = append(params, vv)
			}
			where += fmt.Sprintf("`%v` %v (%v)", key, op, valsStr[1:])
		case "between":
			where += fmt.Sprintf("`%v` %v ? and ?", key, op)
			params = append(params, v[1], v[2])
		case "like":
			where += fmt.Sprintf("`%v` %v ?", key, op)
			params = append(params, fmt.Sprintf("%%%v%%", v[1]))
		case "notLike":
			where += fmt.Sprintf("`%v` not like ?", key)
			params = append(params, fmt.Sprintf("%%%v%%", v[1]))
		case "isNull":
			where += fmt.Sprintf("`%v` is null", key)
		case "notNull":
			where += fmt.Sprintf("`%v` not null", key)
		case "sql":
			where += fmt.Sprint(v[1])
		default:
			where += fmt.Sprintf("`%v` %v ?", key, op)
			params = append(params, v[1])
		}
	}

	if where == "" {
		return " 1 ", params
	}
	return where, params
}
