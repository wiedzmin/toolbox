package env

import "github.com/mediocregopher/radix/v3"

func GetRedisValue(key string, client *radix.Pool) ([]byte, *radix.Pool, error) {
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
	return result, client, nil
}
