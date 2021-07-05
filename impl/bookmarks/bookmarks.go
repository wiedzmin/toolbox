package bookmarks

import (
	"encoding/json"

	"github.com/wiedzmin/toolbox/impl/redis"
	"go.uber.org/zap"
)

type Webjump struct {
	URL     string `json:"url"`
	Browser string `json:"browser"`
	VPN     string `json:"vpn"`
}

type Webjumps struct {
	data   []byte
	parsed map[string]Webjump
}

type SearchEngine struct {
	URL     string `json:"url"`
	Browser string `json:"browser"`
	VPN     string `json:"vpn"`
}

type SearchEngines struct {
	data   []byte
	parsed map[string]SearchEngine
}

type Bookmark struct {
	Path        string   `json:"path"`
	Tags        []string `json:"tags"`
	Ebooks      bool     `json:"ebooks"`
	Shell       bool     `json:"shell"`
	TmuxSession string   `json:"tmux"`
}

type Bookmarks struct {
	data   []byte
	parsed map[string]Bookmark
}

var (
	logger *zap.Logger
	r      *redis.Client
)

func init() {
	var err error
	r, err = redis.NewRedisLocal()
	if err != nil {
		l := logger.Sugar()
		l.Fatalw("[init]", "failed connecting to Redis", err)
	}
}

func NewWebjumps(data []byte) (*Webjumps, error) {
	var result Webjumps
	result.data = data
	err := json.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func WebjumpsFromRedis(key string) (*Webjumps, error) {
	webjumpsData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewWebjumps(webjumpsData)
}

func (j *Webjumps) Keys() []string {
	var result []string
	for key := range j.parsed {
		result = append(result, key)
	}
	return result
}

func (j *Webjumps) Get(key string) *Webjump {
	url, ok := j.parsed[key]
	if !ok {
		return nil
	}
	return &url
}

func NewSearchEngines(data []byte) (*SearchEngines, error) {
	var result SearchEngines
	result.data = data
	err := json.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func SearchEnginesFromRedis(key string) (*SearchEngines, error) {
	searchenginesData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewSearchEngines(searchenginesData)
}

func (e *SearchEngines) Keys() []string {
	var result []string
	for key := range e.parsed {
		result = append(result, key)
	}
	return result
}

func (e *SearchEngines) Get(key string) *SearchEngine {
	url, ok := e.parsed[key]
	if !ok {
		return nil
	}
	return &url
}

func NewBookmarks(data []byte) (*Bookmarks, error) {
	var result Bookmarks
	result.data = data
	err := json.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func BookmarksFromRedis(key string) (*Bookmarks, error) {
	bookmarksData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewBookmarks(bookmarksData)
}

func (bm *Bookmarks) Keys() []string {
	var result []string
	for key := range bm.parsed {
		result = append(result, key)
	}
	return result
}

func (bm *Bookmarks) Get(key string) *Bookmark {
	url, ok := bm.parsed[key]
	if !ok {
		return nil
	}
	return &url
}
