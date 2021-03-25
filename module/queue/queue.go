package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/adjust/rmq/v4"
	"github.com/go-redis/redis/v8"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type QueueModule struct {
	*module.DefaultModule

	id            string
	queues        map[string]rmq.Queue
	topics        map[string]*topic
	redisClient   *redis.Client
	rmqConn       rmq.Connection
	startConsumer bool
	errChan       chan error
}

type topic struct {
	name           string
	job            Job
	prefetchLimits int64
	pollDuration   time.Duration
	consumerCount  int
}

var queueModule = &QueueModule{
	queues:  make(map[string]rmq.Queue),
	topics:  make(map[string]*topic),
	errChan: make(chan error),
}

func GetQueueModule() *QueueModule {
	return queueModule
}

func SetQueueRedisConfig(config *Config) {
	queueModule.redisClient = getRedisClient(config)
}

func SetQueueRedisClient(client *redis.Client) {
	queueModule.redisClient = client
}

func SetId(id string) {
	queueModule.id = id
}

func StartConsuming() {
	queueModule.startConsumer = true
}

func (this *QueueModule) Init() error {
	if queueModule.redisClient == nil {
		return errors.New("redis config not set")
	}
	var err error
	this.rmqConn, err = rmq.OpenConnectionWithRedisClient(this.id, queueModule.redisClient, this.errChan)
	if err != nil {
		return err
	}
	handlerError(this.errChan)
	for name, topic := range this.topics {
		this.queues[name], err = this.rmqConn.OpenQueue(name)
		if err != nil {
			return err
		}

		if this.startConsumer {
			this.queues[name].StartConsuming(topic.prefetchLimits, topic.pollDuration)
			for i := 0; i < topic.consumerCount; i++ {
				this.queues[name].AddConsumer(name, newJob(name, topic.job))
			}
			this.queues[name].PurgeRejected()
		}
	}

	if this.startConsumer {
		autoReturnRejected()
		cleanWorker()
	}

	logrus.Info("queue module inited")
	return nil
}

func (this *QueueModule) Stop() {
	if !this.startConsumer {
		return
	}
	logrus.Info("Stopping queue")
	wg := &sync.WaitGroup{}
	// 停止消费者继续接收消息
	for _, tpc := range this.topics {
		logrus.Infof("Stopping %v queue", tpc.name)
		closeFinished := this.queues[tpc.name].StopConsuming()
		wg.Add(1)
		go func(closeFinished <-chan struct{}, wg *sync.WaitGroup, topic *topic) {
			<-closeFinished
			wg.Done()
			logrus.Infof("Stopped %v queue", topic.name)
		}(closeFinished, wg, tpc)
	}
	wg.Wait()
	// 关闭清理丢失的连接
	quitClean <- 1
	quitReturnRejected <- 1
	logrus.Info("Stopped queue")
}

// Push Push
func Push(key string, msg interface{}) error {
	taskBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, ok := queueModule.queues[key]; !ok {
		return fmt.Errorf("unknown queue %v", key)
	}
	err = queueModule.queues[key].PublishBytes(taskBytes)
	if err != nil {
		return err
	}
	return nil
}

// Clean Clean
func Clean() error {
	cleaner := rmq.NewCleaner(queueModule.rmqConn)
	_, err := cleaner.Clean()
	if err != nil {
		logrus.WithError(err).Error("failed to clean consumer")
	}
	return err
}

// Status 队列状态
func Status() (rmq.Stats, error) {
	list := []string{}
	for name := range queueModule.queues {
		list = append(list, name)
	}
	return queueModule.rmqConn.CollectStats(list)
}

/*
 * AddTopic 添加Topic
 * @params topicName
 * @params consumer
 * @params prefetchLimits 每次从redis队列中取多少
 * @params consumerCount 消费者数量
 * @params pollDuration redis检测间隔
 */
func AddTopic(topicName string, job Job, prefetchLimits, consumerCount int, pollDuration time.Duration) error {
	queueModule.topics[topicName] = &topic{
		name:           topicName,
		job:            job,
		prefetchLimits: int64(prefetchLimits),
		consumerCount:  consumerCount,
		pollDuration:   pollDuration,
	}

	return nil
}
