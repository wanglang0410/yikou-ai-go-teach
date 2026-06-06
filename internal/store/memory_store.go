package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
	"time"
)

type MemoryStore interface {
	GetMessages(ctx context.Context) ([]*schema.Message, error)
	AppendMessage(ctx context.Context, message *schema.Message) error
	ClearMessages(ctx context.Context) error
}

type RedisMemoryStore struct {
	redisClient       *redis.Client
	memoryId          string
	maxMemoryMessages int
	ttl               time.Duration
}

func NewRedisMemoryStore(redisClient *redis.Client, memoryId string, maxMemoryMessages int, ttl time.Duration) *RedisMemoryStore {
	return &RedisMemoryStore{
		redisClient:       redisClient,
		memoryId:          memoryId,
		maxMemoryMessages: maxMemoryMessages,
		ttl:               ttl,
	}
}

func (r RedisMemoryStore) GetMessages(ctx context.Context) ([]*schema.Message, error) {
	key := fmt.Sprintf("memory:%s", r.memoryId)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return decodeMessagesFromJSON(data)
}

func (r RedisMemoryStore) AppendMessage(ctx context.Context, message *schema.Message) error {
	messages, err := r.GetMessages(ctx)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			messages = []*schema.Message{}
		} else {
			return err
		}
	}
	messages = append(messages, message)
	messagesToJSON, err := encodeMessagesToJSON(messages)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("memory:%s", r.memoryId)
	return r.redisClient.Set(ctx, key, messagesToJSON, r.ttl).Err()
}

func (r RedisMemoryStore) ClearMessages(ctx context.Context) error {
	key := fmt.Sprintf("memory:%s", r.memoryId)
	return r.redisClient.Del(ctx, key).Err()
}

func encodeMessagesToJSON(msgs []*schema.Message) ([]byte, error) {
	return json.Marshal(msgs)
}

func decodeMessagesFromJSON(data []byte) ([]*schema.Message, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var msgs []*schema.Message
	err := json.Unmarshal(data, &msgs)
	return msgs, err
}
