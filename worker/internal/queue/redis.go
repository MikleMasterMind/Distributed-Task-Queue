package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	key    string
}

func NewRedis(url string, key string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &RedisQueue{client: redis.NewClient(opts), key: key}, nil
}

func (q *RedisQueue) Pop(ctx context.Context) (string, error) {
	vals, err := q.client.BRPop(ctx, 0, q.key).Result()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", err
	}
	if len(vals) < 2 {
		return "", errors.New("unexpected BRPop result")
	}
	return vals[1], nil
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}