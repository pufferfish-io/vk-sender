package logger

import (
	"go.uber.org/zap"
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
	l, _ := zap.NewProduction()
	return &ZapLogger{log: l.Sugar()}, func() { _ = l.Sync() }
}

func (z *ZapLogger) Info(msg string, args ...any)  { z.log.Infof(msg, args...) }
func (z *ZapLogger) Debug(msg string, args ...any) { z.log.Debugf(msg, args...) }
func (z *ZapLogger) Error(msg string, args ...any) { z.log.Errorf(msg, args...) }
