package dice

import (
	"bytes"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func setLogger(showDebug bool) {
	if showDebug {
		log, _ = zap.NewDevelopment()
		return
	}
	log, _ = zap.NewProduction()
}

type sqlFilterMarshaler struct {
	f *SQLFilter
}

func (sfm sqlFilterMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("limit", sfm.f.limit)
	enc.AddInt("offset", sfm.f.offset)
	return nil
}

func logToml(contents interface{}) {
	var buf bytes.Buffer
	toml.NewEncoder(&buf).Encode(contents)
	log.Sugar().Debug(string(buf.Bytes()))
}
