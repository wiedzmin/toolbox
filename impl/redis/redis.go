package redis

import (
	"strconv"

	"github.com/mediocregopher/radix/v3"
	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

type Client struct {
	conn *radix.Pool
}

func NewRedisLocal() (*Client, error) {
	l := logger.Sugar()
	pool, err := radix.NewPool("tcp", "127.0.0.1:6379", 1)
	if err != nil {
		l.Warnw("[NewRedisLocal]", "err", err)
		return nil, err
	}
	l.Debugw("[NewRedisLocal]", "pool", pool)
	return &Client{pool}, nil
}

func (r *Client) GetValue(key string) ([]byte, error) {
	l := logger.Sugar()
	var result []byte
	err := r.conn.Do(radix.Cmd(&result, "GET", key))
	if err != nil {
		return nil, err
	}
	l.Debugw("[GetRedisValue]", "key", key, "result", result)
	return result, nil
}

func (r *Client) GetValuesMapFuzzy(pattern string) (map[string][]byte, error) {
	l := logger.Sugar()
	result := make(map[string][]byte)
	sc := radix.NewScanner(r.conn, radix.ScanOpts{Command: "SCAN", Pattern: pattern})
	defer sc.Close()
	var key string
	for sc.Next(&key) {
		value, err := r.GetValue(key)
		if err != nil {
			return nil, err
		}
		l.Debugw("[GetRedisValuesFuzzy]", "key", key, "value", string(value))
		result[key] = value
	}
	return result, nil
}

func (r *Client) SetValue(key, value string) error {
	l := logger.Sugar()
	l.Debugw("[SetRedisValue]", "key", key, "value", value)
	return r.conn.Do(radix.Cmd(nil, "SET", key, value))
}

func (r *Client) DeleteValue(key string) error {
	l := logger.Sugar()
	l.Debugw("[DeleteRedisValue]", "key", key)
	return r.conn.Do(radix.Cmd(nil, "DEL", key))
}

func (r *Client) KeyExists(key string) bool {
	l := logger.Sugar()
	var exists int
	err := r.conn.Do(radix.Cmd(&exists, "EXISTS", key))
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

func (r *Client) AppendToList(key string, value string) error {
	l := logger.Sugar()
	l.Debugw("[PushToList]", "key", key, "value", value)
	return r.conn.Do(radix.Cmd(nil, "RPUSH", key, value))
}

func (r *Client) GetList(key string, offset, limit int) ([]string, error) {
	l := logger.Sugar()
	l.Debugw("[GetList]", "key", key, "offset", offset, "limit", limit)
	var result []string
	err := r.conn.Do(radix.Cmd(&result, "LRANGE", key, strconv.Itoa(offset), strconv.Itoa(limit)))
	if err != nil {
		return nil, err
	}
	return result, nil
}
