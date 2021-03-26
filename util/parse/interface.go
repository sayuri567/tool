package parse

import (
	"reflect"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Interface2Float64 Interface2Float64
func Interface2Float64(v interface{}) float64 {
	switch v.(type) {
	case float64:
		return v.(float64)
	case float32:
		return float64(v.(float32))
	case string:
		f, err := strconv.ParseFloat(v.(string), 64)
		if err != nil {
			logrus.WithError(err).WithField("value", v).Error("failed to parse float64")
		}
		return f
	case int64:
		return float64(v.(int64))
	case int32:
		return float64(v.(int32))
	case int:
		return float64(v.(int))
	default:
		logrus.WithFields(logrus.Fields{"value": v, "valuetype": reflect.TypeOf(v).String()}).Error("unknown type")
	}
	return 0
}

// Interface2Int64 Interface2Int64
func Interface2Int64(v interface{}) int64 {
	switch v.(type) {
	case float64:
		return int64(v.(float64))
	case float32:
		return int64(v.(float32))
	case string:
		f, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			logrus.WithError(err).WithField("value", v).Error("failed to parse int64")
		}
		return f
	case int64:
		return v.(int64)
	case int32:
		return int64(v.(int32))
	case int:
		return int64(v.(int))
	default:
		logrus.WithFields(logrus.Fields{"value": v, "valuetype": reflect.TypeOf(v).String()}).Error("unknown type")
	}
	return 0
}

// Interface2String Interface2String
func Interface2String(v interface{}) string {
	switch v.(type) {
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', 2, 64)
	case float32:
		return strconv.FormatFloat(float64(v.(float32)), 'f', 2, 64)
	case string:
		return v.(string)
	case int64:
		return strconv.FormatInt(v.(int64), 10)
	case int32:
		return strconv.FormatInt(int64(v.(int32)), 10)
	case int:
		return strconv.Itoa(v.(int))
	default:
		logrus.WithFields(logrus.Fields{"value": v, "valuetype": reflect.TypeOf(v).String()}).Error("unknown type")
	}
	return ""
}
