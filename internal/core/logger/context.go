package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

// ToContext возвращает новый контекст, в котором лежит log
func ToContext(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext достаёт логгер из контекста. Если его там нет — возвращает логгер, который ничего не пишет.
func FromContext(ctx context.Context) *zap.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return log
	}
	return zap.NewNop()
}
