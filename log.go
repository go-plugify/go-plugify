package goplugify

import (
	"context"
	"log"
)


var logger Logger

type Logger interface {
	WarnCtx(ctx context.Context, format string, args ...any)
	ErrorCtx(ctx context.Context, format string, args ...any)
	InfoCtx(ctx context.Context, format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Info(format string, args ...any)
}

type DefaultLogger struct{}

func (l *DefaultLogger) WarnCtx(ctx context.Context, format string, args ...any) {
	log.Printf("[WARN] "+format+"\n", args...)
}

func (l *DefaultLogger) ErrorCtx(ctx context.Context, format string, args ...any) {
	log.Printf("[ERROR] "+format+"\n", args...)
}

func (l *DefaultLogger) InfoCtx(ctx context.Context, format string, args ...any) {
	log.Printf("[INFO] "+format+"\n", args...)
}

func (l *DefaultLogger) Warn(format string, args ...any) {
	log.Printf("[WARN] "+format+"\n", args...)
}

func (l *DefaultLogger) Error(format string, args ...any) {
	log.Printf("[ERROR] "+format+"\n", args...)
}

func (l *DefaultLogger) Info(format string, args ...any) {
	log.Printf("[INFO] "+format+"\n", args...)
}
