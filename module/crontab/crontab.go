package crontab

import (
	"reflect"

	"github.com/robfig/cron/v3"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type Crontab struct {
	Spec string
	Cmd  cron.Job
}

type CrontabModule struct {
	*module.DefaultModule

	items   []*Crontab
	crontab *cron.Cron
}

var crontabModule = &CrontabModule{
	items: make([]*Crontab, 0),
}

func GetCrontabModule() *CrontabModule {
	return crontabModule
}

func RegisterCron(crons ...*Crontab) {
	for _, cron := range crons {
		crontabModule.items = append(crontabModule.items, cron)
	}
}

func (m *CrontabModule) Init() error {
	m.crontab = cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(&logger{}), m.skipIfStillRunning()))
	for _, item := range m.items {
		_, err := m.crontab.AddJob(item.Spec, item.Cmd)
		if err != nil {
			return err
		}
	}
	logrus.Info("crontab module inited")
	return nil
}

func (m *CrontabModule) Run() error {
	m.crontab.Start()
	return nil
}

func (m *CrontabModule) Stop() {
	logrus.Info("Stopping crontab")
	m.crontab.Stop()
	logrus.Info("Stopped crontab")
}

func (m *CrontabModule) skipIfStillRunning() cron.JobWrapper {
	return func(j cron.Job) cron.Job {
		var name = reflect.TypeOf(j).String()
		limitCh := make(chan struct{}, 1)
		limitCh <- struct{}{}
		return cron.FuncJob(func() {
			select {
			case v := <-limitCh:
				defer func() { limitCh <- v }()
				j.Run()
			default:
				logrus.WithField("cronName", name).Info("skip crontab")
			}
		})
	}
}
