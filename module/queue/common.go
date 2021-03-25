package queue

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/sayuri567/gorun"
	"github.com/sirupsen/logrus"
)

// Config 配置
type Config struct {
	Address     string
	Database    int
	Password    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout int64
}

var (
	quitClean          = make(chan int)
	quitReturnRejected = make(chan int)
)

func getRedisClient(config *Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Network:     "tcp",
		Addr:        config.Address,
		Password:    config.Password,
		DB:          config.Database,
		MaxRetries:  5,
		PoolSize:    config.MaxActive,
		IdleTimeout: time.Duration(config.IdleTimeout) * time.Second,
	})
}

func cleanWorker() {
	Clean()
	go func() {
		defer gorun.Recover("panic")
		for {
			var timer *time.Timer
			timer = time.NewTimer(time.Hour)
			select {
			case <-timer.C:
				Clean()
			case <-quitClean:
				timer.Stop()
				return
			}
		}
	}()
}

func autoReturnRejected() {
	go func() {
		defer gorun.Recover("panic")
		errorCount := map[string]int{}
		retryCount := map[string]int{}
		// TODO 无法辨别哪条消息错误几次，所以统一重试3次，然后丢弃
		for {
			var timer *time.Timer
			timer = time.NewTimer(10 * time.Second)
			select {
			case <-timer.C:
				for key, q := range queueModule.queues {
					if retryCount[key] > 2 {
						logrus.WithField("msgCount", q.PurgeRejected()).Error("retry 3 times for there messages, pruge it")
						retryCount[key] = 0
						continue
					}
					errorCount[key] = q.ReturnRejected(100)
					if errorCount[key] > 0 {
						retryCount[key]++
						logrus.WithField("msgCount", errorCount[key]).WithField("retryCount", retryCount[key]).Warn("return rejected message to ready")
					} else {
						retryCount[key] = 0
					}
				}
			case <-quitReturnRejected:
				timer.Stop()
				return
			}
		}
	}()
}
