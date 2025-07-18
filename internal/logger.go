package internal

import "go.uber.org/zap"

type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

type ZapLogger struct {
	s *zap.SugaredLogger
}

func NewZapLogger(s *zap.SugaredLogger) *ZapLogger {
	return &ZapLogger{s: s}
}

func (l *ZapLogger) Info(args ...interface{})                  { l.s.Info(args...) }
func (l *ZapLogger) Infof(format string, args ...interface{})  { l.s.Infof(format, args...) }
func (l *ZapLogger) Warn(args ...interface{})                  { l.s.Warn(args...) }
func (l *ZapLogger) Warnf(format string, args ...interface{})  { l.s.Warnf(format, args...) }
func (l *ZapLogger) Error(args ...interface{})                 { l.s.Error(args...) }
func (l *ZapLogger) Errorf(format string, args ...interface{}) { l.s.Errorf(format, args...) }
func (l *ZapLogger) Debug(args ...interface{})                 { l.s.Debug(args...) }
func (l *ZapLogger) Debugf(format string, args ...interface{}) { l.s.Debugf(format, args...) }
func (l *ZapLogger) Fatal(args ...interface{})                 { l.s.Fatal(args...) }
func (l *ZapLogger) Fatalf(format string, args ...interface{}) { l.s.Fatalf(format, args...) }
