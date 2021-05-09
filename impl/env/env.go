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

func GetRedisValuesFuzzy(pattern string, client *radix.Pool) (map[string][]byte, *radix.Pool, error) {
	result := make(map[string][]byte)
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, nil, err
		}
	}
	sc := radix.NewScanner(client, radix.ScanOpts{Command: "SCAN", Pattern: "vpn/*/is_up"})
	defer sc.Close()
	var key string
	for sc.Next(&key) {
		value, _, err := GetRedisValue(key, client)
		if err != nil {
			return nil, nil, err
		}
		result[key] = value
	}
	return result, client, nil
}

func SetRedisValue(key, value string, client *radix.Pool) (*radix.Pool, error) {
	var err error
	if client == nil {
		client, err = radix.NewPool("tcp", "127.0.0.1:6379", 1)
		if err != nil {
			return nil, err
		}
	}
	err = client.Do(radix.Cmd(nil, "SET", key, value))
	if err != nil {
		return nil, err
	}
	return client, nil
}
