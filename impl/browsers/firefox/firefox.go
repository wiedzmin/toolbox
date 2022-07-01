package firefox

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/pierrec/lz4/v4"
	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

type SessionFormat int8

const (
	SESSION_FORMAT_JSON SessionFormat = 0
	SESSION_FORMAT_ORG  SessionFormat = 1
	MOZ_LZ_MAGIC_HEADER               = "mozLz40\x00"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func RawSessionsPath() *string {
	path, err := impl.AtHomedir(".mozilla/firefox/profile.default/sessionstore-backups")
	if err != nil {
		return nil
	}
	return path
}

func GetSessionData(sessionFilename string) ([]byte, error) {
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
		return nil, impl.FileFormatError{fmt.Sprintf("wrong header: %s", string(header))}
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
