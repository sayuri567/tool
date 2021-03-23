package job

import (
	"encoding/json"
	"reflect"

	"github.com/adjust/rmq"
	"github.com/sayuri567/gorun"
	"github.com/sirupsen/logrus"
)

// Job job需要实现的接口
type Job interface {
	// After 收尾工作方法
	// After() error
	// Before 准备工作方法
	// Before() error
	// Program 主方法
	Program() error
}

// BaseJob BaseJob
type BaseJob struct {
	job  Job
	name string
}

// NewJob NewJob
func newJob(name string, job Job) *BaseJob {
	return &BaseJob{
		job:  job,
		name: name,
	}
}

// Consume Consume
func (j *BaseJob) Consume(delivery rmq.Delivery) {
	defer gorun.Recover("panic")
	var isRejected = false
	var param = reflect.New(reflect.TypeOf(j.job).Elem())
	err := json.Unmarshal([]byte(delivery.Payload()), param.Interface())
	if err != nil {
		logrus.WithError(err).WithField("job", j.name).Error("failed to parse json message")
		isRejected = delivery.Reject()
		return
	}
	before := param.MethodByName("Before")
	if before.IsValid() && !before.IsNil() && before.Kind() == reflect.Func {
		beforeErr := before.Call(nil)
		if beforeErr != nil && len(beforeErr) > 0 && !beforeErr[0].IsNil() {
			logrus.WithField("error", beforeErr[0].Interface()).WithField("job", j.name).Error("run job before function has error")
			// 执行准备工作方法时出错，驳回消息，尝试重试
			isRejected = delivery.Reject()
			return
		}
	}
	program := param.MethodByName("Program")
	if program.Kind() != reflect.Func {
		logrus.WithError(err).WithField("job", j.name).Error("job has no program function")
		// 找不到Program方法，驳回消息
		isRejected = delivery.Reject()
		return
	}
	programErr := program.Call(nil)
	if programErr != nil && len(programErr) > 0 && !programErr[0].IsNil() {
		logrus.WithField("error", programErr[0].Interface()).WithField("job", j.name).Error("run job program function has error")
		// 执行Program出错，驳回消息，并执行after方法
		isRejected = delivery.Reject()
	}

	after := param.MethodByName("After")
	if after.IsValid() && !after.IsNil() && after.Kind() == reflect.Func {
		afterErr := after.Call(nil)
		if afterErr != nil && len(afterErr) > 0 && !afterErr[0].IsNil() {
			// 执行After出错，驳回消息，并执行after方法
			logrus.WithField("error", afterErr[0].Interface()).WithField("job", j.name).Error("run job after function has error")
		}
	}

	if !isRejected {
		// 消息没有驳回，成功消费
		delivery.Ack()
	}
}
