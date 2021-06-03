package baidupan

import "sync"

type Config struct {
	AppId     string
	AppKey    string
	SecretKey string
	SignKey   string
}

type BaiduPan struct {
	config *Config
}

type BaiduPanHandle func() error

var (
	baiduPan = &BaiduPan{}
	configs  = map[string]*Config{}
	rwlock   = sync.RWMutex{}
)

func GetBaiduPan(key string) *BaiduPan {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return &BaiduPan{config: configs[key]}
}

func SetConfig(key string, config *Config) {
	rwlock.Lock()
	defer rwlock.Unlock()
	configs[key] = config
}
