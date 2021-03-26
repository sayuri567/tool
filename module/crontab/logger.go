package crontab

import "github.com/sirupsen/logrus"

type logger struct{}

func (log *logger) Error(err error, msg string, keysAndValues ...interface{}) {
	logrus.WithError(err).WithField("info_fields", keysAndValues).Error(msg)
}

func (log *logger) Info(msg string, keysAndValues ...interface{}) {
	logrus.WithField("info_fields", keysAndValues).Info(msg)
}
