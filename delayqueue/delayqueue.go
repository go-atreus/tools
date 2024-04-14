package delayqueue

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"strconv"
	"time"
)

type DelayQueue interface {
	StartWorker(ctx context.Context)
	// PublishDelayTask publish a task to delay queue
	PublishDelayTask(queue string, payload any, delaySecond int)
}

var (
	dq *delayQueue
)

type Job struct {
	DeliveryTs    int    `json:"delivery-ts" msgpack:"deliveryTs"`
	Retry         int    `json:"retry" msgpack:"retry"`
	ExecEndTime   string `json:"execEndTime" msgpack:"execEndTime"`
	Msg           string `json:"msg" msgpack:"msg"`
	ExecTs        int    `json:"execTs" msgpack:"execTs"`
	ExecStartTime string `json:"execStartTime" msgpack:"execStartTime"`
	Status        int    `json:"status" msgpack:"status"`
	Timeout       int    `json:"timeout" msgpack:"timeout"`
	ExecNum       int    `json:"execNum" msgpack:"execNum"`
	Payload       []byte `json:"payload" msgpack:"payload"`
	PublishTime   string `json:"publishTime" msgpack:"publishTime"`
	DeliveryTime  string `json:"deliveryTime" msgpack:"deliveryTime"`
}

func PublishDelayTask(queue string, payload any, delaySecond int) (err error) {
	if dq.RedisCli == nil {
		err = errors.New("redis client not init, please use WithRedis() to init")
		return
	}
	dq.PublishDelayTask(queue, payload, delaySecond)
	return
}

type QueueCallback func(payload []byte) bool

type delayQueue struct {
	QueueName  string
	Callback   QueueCallback
	RedisCli   redis.Cmdable
	Concurrent int
	Serializer Serializer
}

func (d delayQueue) StartWorker(ctx context.Context) {
	defer ants.Release()

	workerQueue := d.QueueName + "worker"
	// 注册延迟队列
	d.RegisterDelayQueue(ctx, d.QueueName, workerQueue)
	// master线程：分发到期的延迟任务到worker队列
	masterFunc := func() {
		ticker := time.NewTicker(time.Second * 2)
		for {
			select {
			case <-ticker.C:
				// 获取延迟任务
				tasks := d.getDelayTask(ctx, d.QueueName)
				for _, taskId := range tasks {
					//rem := d.RedisCli.ZRem(ctx, d.QueueName, taskId)
					//fmt.Println("get delay task: ", taskId, rem)
					lockKey := "delivery-" + taskId
					lock := d.setLock(lockKey, "delivery-task", 60)
					if lock != "OK" {
						fmt.Println("到期的任务正在被推到处理队列", taskId)
						continue
					}
					job, err := d.getJob(ctx, taskId)
					// 当操作redis失败或没有job处理时，等待后在执行。
					if (err != nil && err != redis.Nil) || job == nil {
						continue
					}
					// ready状态的任务，需推送到worker队列，然后等待ack（unack的不能重复推送）
					// 如果worker线程执行过程中，进程被kill或重启，可能会导致ack发送不了从而任务就不会再执行了，为防止该情况的发生，需加上超时判断（unack的任务，如超时会继续往worker-q中推送）
					// status=1只可能有3种情况，1（正常）：未到调度周期；2（正常）：任务被推送到woker-q后未执行完成；3（不正常）：执行任务时出现异常，比如redis 连接或读取超时
					// 分发到worker队列
					if job.Status == 0 {
						_ = d.RedisCli.RPush(ctx, workerQueue, taskId)
						job.Status = 1
						marshal, _ := Marshal(job)
						_ = d.RedisCli.Set(ctx, taskId, marshal, 0)
					}

				}
			}
		}
	}
	// worker线程：用来处理延迟任务
	workerFunc := func() {
		ticker := time.NewTicker(time.Second)
		tickerDone := make(chan struct{}, 1)
		tickerDone <- struct{}{}
		idsChan := make(chan string, 1)
		go func() {
			for {
				select {
				case <-ticker.C:
					<-tickerDone
					// 获取worker任务
					pop := d.RedisCli.LPop(ctx, workerQueue)
					if pop.Err() == nil {
						fmt.Println("get worker task: ", pop.Val())
						idsChan <- pop.Val()
					}
					tickerDone <- struct{}{}
				}
			}
		}()
		for {
			select {
			case ids := <-idsChan:
				// 处理worker任务
				fmt.Println("process work task: ", ids)
				job, err := d.getJob(ctx, ids)
				if err != nil {
					fmt.Println("process worker task error: ", ids, err)
					continue
				}
				result := d.Callback(job.Payload)
				if result {
					d.delDelayTask(ctx, ids)
				}

			}

		}
	}
	pool, err := ants.NewPool(d.Concurrent + 1)
	if err != nil {
		panic(err)
	}
	for i := 0; i < d.Concurrent; i++ {
		err := pool.Submit(workerFunc)
		if err != nil {
			panic(err)
		}
	}
	err = pool.Submit(masterFunc)
	if err != nil {
		panic(err)
	}
}

func (d delayQueue) PublishDelayTask(queue string, payload any, delaySecond int) {
	taskUuid := queue + uuid.New().String()
	score := time.Now().UnixMilli() + int64(delaySecond*1000)
	d.RedisCli.ZAdd(context.Background(), queue, &redis.Z{Score: float64(score), Member: taskUuid})
	var err error
	inner, err := Marshal(payload)
	if err != nil {
		panic(err)
	}
	job := &Job{
		Payload:     inner,
		Status:      0,
		PublishTime: time.Now().Format("2006-01-02 15:04:05"),
		ExecNum:     0,
	}
	bytes, err := Marshal(job)
	if err != nil {
		panic(err)
	}
	d.RedisCli.Set(context.Background(), taskUuid, bytes, 0)
	fmt.Println("publish delay task: ", taskUuid, score)
}

func (d delayQueue) setLock(keyName string, value string, timeout int) string {
	ctx := context.Background()
	result := d.RedisCli.SetEX(ctx, keyName, value, time.Duration(timeout)*time.Second).Val()
	if result != "OK" {
		return d.RedisCli.Get(ctx, keyName).Val()
	}
	return "OK"
}

func (d delayQueue) getDelayTask(ctx context.Context, queueName string) []string {
	milli := time.Now().UnixMilli()
	score := d.RedisCli.ZRangeByScore(ctx, queueName, &redis.ZRangeBy{Min: "0", Max: strconv.FormatInt(milli, 10)})
	return score.Val()
}

// RegisterDelayQueue register delay queue
// 注册延迟队列
// 队列名称 name
// worker: worker name
func (d delayQueue) RegisterDelayQueue(ctx context.Context, name string, worker string) {
	d.RedisCli.HSet(ctx, "all-delay-q-set", name, worker)
}

func (d delayQueue) delDelayTask(ctx context.Context, ids string) {
	rem := d.RedisCli.ZRem(ctx, d.QueueName, ids)
	fmt.Println("zrem: ", ids, rem.Val())
	del := d.RedisCli.Del(ctx, ids)
	fmt.Println("del: ", ids, del.Val())
}

func (d delayQueue) getJob(ctx context.Context, id string) (*Job, error) {
	ret := d.RedisCli.Get(ctx, id)
	if err := ret.Err(); err != nil {
		return nil, err
	}
	job := &Job{}
	value, err := ret.Bytes()
	if err != nil {
		return nil, err
	}
	err = Unmarshal(value, job)
	if err != nil {
		return nil, err
	}
	return job, err
}

func NewQueue(ctx context.Context, queueName string, redisCli redis.Cmdable, callback QueueCallback, opts ...Option) (DelayQueue, error) {
	newDq := delayQueue{QueueName: queueName, RedisCli: redisCli, Callback: callback}
	//循环调用opts
	for _, option := range opts {
		option(&newDq)
	}
	dq = &newDq
	dq.StartWorker(ctx)
	return &newDq, nil
}
