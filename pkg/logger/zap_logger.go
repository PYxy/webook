package logger

import "go.uber.org/zap"

// 适配器模式
// zap 的接口 适配 自定义接口ZapLogger
type ZapLogger struct {
	l *zap.Logger
}

func NewZaplogger(l *zap.Logger) *ZapLogger {
	return &ZapLogger{
		l: l,
	}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	////TODO implement me
	//panic("implement me")
	z.l.Debug(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	//TODO implement me
	z.l.Info(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	//TODO implement me
	z.l.Warn(msg, z.toZapFields(args)...)

}

func (z *ZapLogger) Error(msg string, args ...Field) {
	//TODO implement me
	z.l.Error(msg, z.toZapFields(args)...)

}

func (z *ZapLogger) toZapFields(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		res = append(res, zap.Any(arg.Key, arg.Value))
	}
	return res
}
