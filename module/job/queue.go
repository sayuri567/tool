package job

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/adjust/rmq"
	"github.com/go-redis/redis"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type QueueManager struct {
	*module.DefaultModule

	id            string
	queues        map[string]rmq.Queue
	topics        map[string]*topic
	redisClient   *redis.Client
	rmqConn       rmq.Connection
	startConsumer bool
}

type topic struct {
	name           string
	job            Job
	prefetchLimits int
	pollDuration   time.Duration
	consumerCount  int
}

var queueManager = &QueueManager{
	queues: make(map[string]rmq.Queue),
	topics: make(map[string]*topic),
}

func GetQueueManager() *QueueManager {
	return queueManager
}

func SetQueueRedisConfig(config *Config) {
	queueManager.redisClient = getQueueClient(config)
}

func SetQueueRedisClient(client *redis.Client) {
	queueManager.redisClient = client
}

func SetId(id string) {
	queueManager.id = id
}

func StartConsuming() {
	queueManager.startConsumer = true
}

func (this *QueueManager) Init() error {
	if queueManager.redisClient == nil {
		return errors.New("redis config not set")
	}
	this.rmqConn = rmq.OpenConnectionWithRedisClient(this.id, queueManager.redisClient)
	for name, topic := range this.topics {
		this.queues[name] = this.rmqConn.OpenQueue(name)

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

	return nil
}

func (this *QueueManager) Stop() {
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
	if _, ok := queueManager.queues[key]; !ok {
		return fmt.Errorf("unknown queue %v", key)
	}
	res := queueManager.queues[key].PublishBytes(taskBytes)
	if res == false {
		return fmt.Errorf("failed to push message queue")
	}
	return nil
}

// Clean Clean
func Clean() error {
	cleaner := rmq.NewCleaner(queueManager.rmqConn)
	err := cleaner.Clean()
	if err != nil {
		logrus.WithError(err).Error("failed to clean consumer")
	}
	return err
}

// Status 队列状态
func Status() rmq.Stats {
	list := []string{}
	for name := range queueManager.queues {
		list = append(list, name)
	}
	return queueManager.rmqConn.CollectStats(list)
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
	queueManager.topics[topicName] = &topic{
		name:           topicName,
		job:            job,
		prefetchLimits: prefetchLimits,
		consumerCount:  consumerCount,
		pollDuration:   pollDuration,
	}

	return nil
}
