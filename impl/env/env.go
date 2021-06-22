package env

import (
	"github.com/mediocregopher/radix/v3"
	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func GetRedisValue(key string, client *radix.Pool) ([]byte, *radix.Pool, error) {
	l := logger.Sugar()
	var result []byte
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, nil, err
		}
	}
	err = client.Do(radix.Cmd(&result, "GET", key))
	if err != nil {
		return nil, nil, err
	}
	l.Debugw("[GetRedisValue]", "key", key, "result", result)
	return result, client, nil
}

func GetRedisValuesFuzzy(pattern string, client *radix.Pool) (map[string][]byte, *radix.Pool, error) {
	l := logger.Sugar()
	result := make(map[string][]byte)
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, nil, err
		}
	}
	sc := radix.NewScanner(client, radix.ScanOpts{Command: "SCAN", Pattern: pattern})
	defer sc.Close()
	var key string
	for sc.Next(&key) {
		value, _, err := GetRedisValue(key, client)
		if err != nil {
			return nil, nil, err
		}
		l.Debugw("[GetRedisValuesFuzzy]", "key", key, "value", string(value))
		result[key] = value
	}
	return result, client, nil
}

func SetRedisValue(key, value string, client *radix.Pool) (*radix.Pool, error) {
	l := logger.Sugar()
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, err
		}
	}
	l.Debugw("[SetRedisValue]", "key", key, "value", value)
	err = client.Do(radix.Cmd(nil, "SET", key, value))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func DeleteRedisValue(key string, client *radix.Pool) (*radix.Pool, error) {
	l := logger.Sugar()
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, err
		}
	}
	l.Debugw("[DeleteRedisValue]", "key", key)
	err = client.Do(radix.Cmd(nil, "DEL", key))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func RedisKeyExists(key string, client *radix.Pool) bool {
	l := logger.Sugar()
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return false
		}
	}

	var exists int
	err = client.Do(radix.Cmd(&exists, "EXISTS", key))
	l.Debugw("[RedisKeyExists]", "key", key, "exists", exists)
	if err == nil {
		switch exists {
		case 0:
			return false
		case 1:
			return true
		}
	}
	return false
}
