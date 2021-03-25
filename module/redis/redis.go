package redis

import (
	"errors"
	"time"

	redigo "github.com/garyburd/redigo/redis"
	"github.com/sayuri567/tool/module"
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

type RedisModule struct {
	*module.DefaultModule

	pools       map[string]*redigo.Pool
	configs     map[string]*Config
	defaultPool string
}

var redisModule = &RedisModule{
	pools:   make(map[string]*redigo.Pool),
	configs: make(map[string]*Config),
}

func GetRedisModule() *RedisModule {
	return redisModule
}

func RegisterRedis(name string, config *Config) {
	if len(redisModule.defaultPool) == 0 {
		redisModule.defaultPool = name
	}
	redisModule.configs[name] = config
}

func Get() redigo.Conn {
	return redisModule.pools[redisModule.defaultPool].Get()
}

func GetConn(name string) redigo.Conn {
	if pool, ok := redisModule.pools[name]; ok {
		return pool.Get()
	}
	return nil
}

func (this *RedisModule) Init() error {
	var err error
	if len(this.configs) == 0 {
		return errors.New("redis config not set")
	}
	for name, config := range this.configs {
		this.pools[name] = this.newPool(config)
		conn := this.pools[name].Get()
		err = conn.Err()
		conn.Close()
		if err != nil {
			break
		}
	}

	if err != nil {
		return err
	}

	logrus.Info("redis module inited")
	return nil
}

func (this *RedisModule) Stop() {
	logrus.Info("Stopping redis connects")
	for name, pool := range this.pools {
		err := pool.Close()
		if err != nil {
			logrus.WithError(err).Errorf("Stop %v redis failed", name)
		}
	}
	logrus.Info("Stopped redis connects")
}

func (this *RedisModule) newPool(config *Config) *redigo.Pool {
	return &redigo.Pool{
		MaxIdle:     config.MaxIdle,
		MaxActive:   config.MaxActive,
		IdleTimeout: time.Duration(config.IdleTimeout) * time.Second,
		Wait:        true,
		Dial: func() (redigo.Conn, error) {
			c, err := redigo.Dial("tcp", config.Address)
			if err != nil {
				logrus.WithField("error", err.Error()).Error("failed to connect redis")
				return nil, err
			}
			if config.Password != "" {
				if _, err := c.Do("AUTH", config.Password); err != nil {
					logrus.WithField("error", err.Error()).Error("failed to auth redis")
					c.Close()
					return nil, err
				}
			}
			c.Do("SELECT", config.Database)
			return c, err
		},
		TestOnBorrow: func(c redigo.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}
