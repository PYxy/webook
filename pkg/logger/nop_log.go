package logger

type NoOpLogger struct {
}

func NewNoOpLogger() LoggerV1 {
	return &NoOpLogger{}
}

func (n *NoOpLogger) Debug(msg string, args ...Field) {}

func (n *NoOpLogger) Info(msg string, args ...Field) {
}

func (n *NoOpLogger) Warn(msg string, args ...Field) {
}

func (n *NoOpLogger) Error(msg string, args ...Field) {
}
