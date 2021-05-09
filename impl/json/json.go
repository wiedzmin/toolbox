package json

import (
	"strings"

	"github.com/Jeffail/gabs"
)

func GetMapByPath(data []byte, path string) (map[string]*gabs.Container, error) {
	dataParsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}
	var entries map[string]*gabs.Container
	if path == "" {
		entries, err = dataParsed.S().ChildrenMap()
	} else {
		entries, err = dataParsed.S(strings.Split(path, ".")...).ChildrenMap()
	}
	return entries, err
}

func StringifyFlatMap(object *gabs.Container) (map[string]string, error) {
	result := make(map[string]string)
	items, err := object.S().ChildrenMap()
	if err != nil {
		return nil, err
	}
	for key, val := range items {
		valStr, ok := val.Data().(string)
		if !ok {
			continue
		}
		result[key] = valStr
	}
	return result, nil
}
