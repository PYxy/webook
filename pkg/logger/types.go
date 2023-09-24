package logger

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)

	Warn(msg string, args ...any)

	Error(msg string, args ...any)
}

func LoggerExample() {
	var l Logger

	phone := "123456789123"
	l.Info("用户未注册,手机号码 %s %d %v %w", phone)
}

type LoggerV1 interface {
	Debug(msg string, args ...Field)
	Info(msg string, args ...Field)

	Warn(msg string, args ...Field)

	Error(msg string, args ...Field)
}

type Field struct {
	Key   string
	Value any
}

func LoggerV1Example() {
	var l Logger

	phone := "123456789123"
	l.Info("用户未注册", Field{
		Key:   "phone",
		Value: phone,
	})
}

type LoggerV2 interface {
	//args 必须是偶数 并且是 key-val key-val
	Debug(msg string, args ...any)
	Info(msg string, args ...any)

	Warn(msg string, args ...any)

	Error(msg string, args ...any)
}

func LoggerV2Example() {
	var l Logger

	phone := "123456789123"
	l.Info("用户未注册", "phone", phone)
}
