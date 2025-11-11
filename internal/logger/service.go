package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

type ZapLogger struct {
	log *zap.SugaredLogger
}

func NewZapLogger() (*ZapLogger, func()) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = localISOTimeEncoder
	l, _ := cfg.Build()
	return &ZapLogger{log: l.Sugar()}, func() { _ = l.Sync() }
}

func localISOTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.In(time.Local).Format(time.RFC3339))
}

func (z *ZapLogger) Info(msg string, args ...any)  { z.log.Infof(msg, args...) }
func (z *ZapLogger) Debug(msg string, args ...any) { z.log.Debugf(msg, args...) }
func (z *ZapLogger) Error(msg string, args ...any) { z.log.Errorf(msg, args...) }
