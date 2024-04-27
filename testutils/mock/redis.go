package mock

import (
	"context"
	"time"
)

type RedisGetCall struct {
	Key string
}

type RedisSetWithTTLErrCall struct {
	Key   string
	Value string
	TTL   time.Duration
}

type MockRedis struct {
	GetCalls           []RedisGetCall
	SetWithTTLErrCalls []RedisSetWithTTLErrCall
	GetDelCalls        []RedisGetCall
	setWithTTLErrErr   error
	KeyValues          map[string]string
}

func NewMockRedis(keyValues map[string]string) *MockRedis {
	return &MockRedis{
		GetCalls:           []RedisGetCall{},
		SetWithTTLErrCalls: []RedisSetWithTTLErrCall{},
		GetDelCalls:        []RedisGetCall{},
		setWithTTLErrErr:   nil,
		KeyValues:          keyValues,
	}
}

func (mock *MockRedis) Get(ctx context.Context, key string) (string, bool) {
	mock.GetCalls = append(mock.GetCalls, RedisGetCall{Key: key})

	value, ok := mock.KeyValues[key]

	return value, ok
}

func (mock *MockRedis) SetWithTTLErr(ctx context.Context, key string, value string, ttl time.Duration) error {
	mock.SetWithTTLErrCalls = append(mock.SetWithTTLErrCalls, RedisSetWithTTLErrCall{
		Key:   key,
		Value: value,
		TTL:   ttl,
	})

	return mock.setWithTTLErrErr
}

func (mock *MockRedis) GetDel(ctx context.Context, key string) (string, bool) {
	mock.GetDelCalls = append(mock.GetDelCalls, RedisGetCall{Key: key})

	value, ok := mock.KeyValues[key]
	if ok {
		delete(mock.KeyValues, key)
	}

	return value, ok
}

func (mock *MockRedis) ConfigureSetWithTTLErr(err error) *MockRedis {
	mock.setWithTTLErrErr = err

	return mock
}
