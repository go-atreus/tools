package delayqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func callback(payload []byte) bool {
	var res OrderInfo
	err := Unmarshal(payload, &res)
	if err != nil {
		fmt.Println(err)
		return false
	}
	marshal, err := json.Marshal(res)
	fmt.Println("callback: ", string(marshal))
	return true
}

type OrderInfo struct {
	OrderId string `json:"order_id"`
	UserId  string `json:"user_id"`
}

func TestQueue(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Username: "",
		Password: "test", // no password set
		DB:       0,      // use default DB
	})
	opts := []Option{
		WithConcurrent(10),
	}
	ctx := context.Background()
	queueName := "order-delay-queue"
	_, err := NewQueue(ctx, queueName, rdb, callback, opts...)
	if err != nil {
		t.Fatal(err)
	}
	order := &OrderInfo{
		OrderId: "123456",
		UserId:  "123456",
	}
	err = PublishDelayTask(queueName, order, 1)
	if err != nil {
		t.Fatal(err)
	}
	ticker := time.NewTicker(time.Second * 3)
	for {
		select {
		case <-ticker.C:
			//err = PublishDelayTask(queueName, "payload", 1)
			//if err != nil {
			//	t.Fatal(err)
			//}
		}
	}

	select {
	case <-ctx.Done():
	}
}
