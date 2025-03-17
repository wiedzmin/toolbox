package firefox

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/pierrec/lz4/v4"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"go.uber.org/zap"
)

type SessionFormat int8

const (
	SESSION_FORMAT_JSON     SessionFormat = 0
	SESSION_FORMAT_ORG      SessionFormat = 1
	SESSION_FORMAT_ORG_FLAT SessionFormat = 2
	MOZ_LZ_MAGIC_HEADER                   = "mozLz40\x00"
	SessionstoreSubpath                   = ".mozilla/firefox/profile.default/sessionstore-backups"
)

var logger *zap.Logger

type SessionLayout struct {
	Windows []Window `json:"windows"`
}

type Window struct {
	Tabs []Tab `json:"tabs"`
}

type Tab struct {
	History []Page `json:"entries"`
}

type Page struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	OriginalURI string `json:"originalURI"`
}

func init() {
	logger = impl.NewLogger()
}

// RawSessionsPath returns path where raw jsonlz4 sessions are stored
func RawSessionsPath() *string {
	path, err := fs.AtHomedir(SessionstoreSubpath)
	if err != nil {
		return nil
	}
	return path
}

// getSessionData returns decompressed session data, given path to "jsonlz4"-compressed session
func getSessionData(sessionFilename string) ([]byte, error) {
	sessionFile, err := os.Open(sessionFilename)
	if err != nil {
		return nil, err
	}
	fi, err := sessionFile.Stat()
	if err != nil {
		return nil, err
	}
	srcContentSize := fi.Size() - int64(len(MOZ_LZ_MAGIC_HEADER)) - 4

	header := make([]byte, len(MOZ_LZ_MAGIC_HEADER))
	_, err = sessionFile.Read(header)
	if err != nil {
		return nil, err
	}
	if string(header) != MOZ_LZ_MAGIC_HEADER {
		return nil, impl.FileFormatError{Content: fmt.Sprintf("wrong header: %s", string(header))}
	}

	dstSizeBytes := make([]byte, 4)
	sessionFile.Read(dstSizeBytes)
	dstSize := binary.LittleEndian.Uint32(dstSizeBytes)

	srcData := make([]byte, srcContentSize)
	sessionFile.Read(srcData)

	dstData := make([]byte, dstSize)
	_, err = lz4.UncompressBlock(srcData, dstData)
	if err != nil {
		return nil, err
	}

	return dstData, nil
}

// LoadSession is used for loading, decompressing and unmarshalling session data
func LoadSession(path string) (*SessionLayout, error) {
	l := logger.Sugar()
	l.Debugw("[LoadSession]", "path", path)
	var session SessionLayout
	deflated, err := getSessionData(path)
	if err != nil {
		return nil, err
	}
	err = jsoniter.Unmarshal(deflated, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DumpSession dumps session data in one of predefined formats
func DumpSession(path string, data *SessionLayout, format SessionFormat, rawUrls, withHistory bool) error {
	l := logger.Sugar()
	l.Debugw("[DumpSession]", "path", path, "data", data, "format", format, "withHistory", withHistory)
	if data == nil {
		return fmt.Errorf("empty session")
	}
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	switch format {
	case SESSION_FORMAT_JSON: // NOTE: tab history dropping is not yet implemented here
		b, err := jsoniter.Marshal(data)
		if err != nil {
			return err
		}
		_, err = writer.Write(b)
		if err != nil {
			return err
		}
	// FIXME: try to generalize Tridactyl workaround(s) below
	case SESSION_FORMAT_ORG:
		index := 1
		var result []string
		for _, w := range data.Windows {
			result = append(result, (fmt.Sprintf("* window %d", index)))
			for _, t := range w.Tabs {
				orgStars := "**"
				for _, p := range t.History {
					l.Debugw("[DumpSession]", "url", p.URL)
					// NOTE: workaround for pasting url into fresh Tridactyl window
					if strings.HasPrefix(p.URL, "moz-extension") {
						continue
					}
					var urlData string
					if rawUrls {
						urlData = p.URL
					} else {
						urlData = fmt.Sprintf("[[%s][%s]]", p.URL, p.Title)
					}
					result = append(result, (fmt.Sprintf("%s %s", orgStars, urlData)))
					if !withHistory {
						l.Debugw("[DumpSession]", "warning", "dropped history")
						break
					} else {
						orgStars = "***"
					}
				}
			}
			index = index + 1
		}
		for _, line := range result {
			_, _ = writer.WriteString(fmt.Sprintf("%s\n", line))
		}
	case SESSION_FORMAT_ORG_FLAT:
		var result []string
		for _, w := range data.Windows {
			for _, t := range w.Tabs {
				orgStars := "*"
				for _, p := range t.History {
					l.Debugw("[DumpSession]", "url", p.URL)
					// NOTE: workaround for pasting url into fresh Tridactyl window
					if strings.HasPrefix(p.URL, "moz-extension") {
						continue
					}
					var urlData string
					if rawUrls {
						urlData = p.URL
					} else {
						urlData = fmt.Sprintf("[[%s][%s]]", p.URL, p.Title)
					}
					result = append(result, (fmt.Sprintf("%s %s", orgStars, urlData)))
					if !withHistory {
						l.Debugw("[DumpSession]", "warning", "dropped history")
						break
					} else {
						orgStars = "**"
					}
				}
			}
		}
		for _, line := range result {
			_, _ = writer.WriteString(fmt.Sprintf("%s\n", line))
		}
	}
	return nil
}
