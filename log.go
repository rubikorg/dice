package dice

import (
	"go.uber.org/zap"
)

var log *zap.Logger

func setLogger(showDebug bool) {
	if showDebug {
		log, _ = zap.NewDevelopment()
		return
	}
	log, _ = zap.NewProduction()
}

// type sqlFilterMarshaler struct {
// 	f *SQLFilter
// }

// func (sfm sqlFilterMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
// 	enc.AddInt("limit", sfm.f.limit)
// 	enc.AddInt("offset", sfm.f.offset)
// 	return nil
// }
