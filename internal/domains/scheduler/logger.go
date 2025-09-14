package scheduler

import (
	"github.com/hibiken/asynq"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// AsynqLogger wraps our logger to implement asynq.Logger interface
type AsynqLogger struct {
	logger *Logger.Logger
}

// NewAsynqLogger creates a new asynq logger wrapper
func NewAsynqLogger(logger *Logger.Logger) asynq.Logger {
	return &AsynqLogger{logger: logger}
}

// Debug implements asynq.Logger.Debug
func (l *AsynqLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

// Info implements asynq.Logger.Info
func (l *AsynqLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

// Warn implements asynq.Logger.Warn
func (l *AsynqLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

// Error implements asynq.Logger.Error
func (l *AsynqLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

// Fatal implements asynq.Logger.Fatal
func (l *AsynqLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}
