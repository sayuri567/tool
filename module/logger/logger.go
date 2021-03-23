package logger

import (
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Level        string
	LogFile      string
	MaxRemainCnt int
	RotationTime time.Duration
	TimeFormat   string
	ExtendFields map[string]string
}

// DefaultFieldHook DefaultFieldHook
type DefaultFieldHook struct {
	*module.DefaultModule
	config    *Config
	formatter logrus.Formatter
}

func NewLogHookModule(config Config) *DefaultFieldHook {
	if len(config.TimeFormat) == 0 {
		config.TimeFormat = "2006-01-02T15:04:05-07:00"
	}
	if len(config.LogFile) > 0 {
		if config.RotationTime < time.Minute {
			config.RotationTime = time.Hour * 24
		}
		if config.MaxRemainCnt < 1 {
			config.MaxRemainCnt = 10
		}
	}
	if len(config.Level) == 0 {
		config.Level = logrus.DebugLevel.String()
	}
	return &DefaultFieldHook{config: &config}
}

// Levels Levels
func (this *DefaultFieldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire Fire
func (this *DefaultFieldHook) Fire(e *logrus.Entry) error {
	for key, value := range this.config.ExtendFields {
		if _, ok := e.Data[key]; !ok {
			e.Data[key] = value
		}
	}
	return nil
}

// InitLogger InitLogger
func (this *DefaultFieldHook) Init() error {
	// 初始化logger
	this.formatter = &logrus.JSONFormatter{
		TimestampFormat: this.config.TimeFormat,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "@time",
			logrus.FieldKeyLevel: "@level",
			logrus.FieldKeyMsg:   "message",
		},
	}
	logrus.ErrorKey = "@error"
	logrus.SetFormatter(this.formatter)
	logLevel := this.config.Level
	if logLevel == "" {
		logLevel = logrus.DebugLevel.String()
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	logrus.SetReportCaller(true)
	logrus.SetLevel(level)
	logrus.AddHook(this)
	if len(this.config.LogFile) > 0 {
		hook, err := this.newLfsHook()
		if err != nil {
			return err
		}
		logrus.AddHook(hook)
	}
	logrus.Info("logger module inited")
	return nil
}

func (this *DefaultFieldHook) newLfsHook() (logrus.Hook, error) {
	writer, err := rotatelogs.New(
		this.config.LogFile+".%Y%m%d%H",
		// WithLinkName 为最新的日志建立软连接，以方便随时找到当前日志文件
		rotatelogs.WithLinkName(this.config.LogFile),

		// WithRotationTime 设置日志分割的时间，这里设置为一小时分割一次
		rotatelogs.WithRotationTime(this.config.RotationTime),

		// WithMaxAge和WithRotationCount 二者只能设置一个，
		// WithMaxAge 设置文件清理前的最长保存时间，
		// WithRotationCount 设置文件清理前最多保存的个数。
		//rotatelogs.WithMaxAge(time.Hour*24),
		rotatelogs.WithRotationCount(uint(this.config.MaxRemainCnt)),
	)

	if err != nil {
		logrus.Errorf("config local file system for logger error: %v", err)
		return nil, err
	}

	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, this.formatter)

	return lfsHook, nil
}
