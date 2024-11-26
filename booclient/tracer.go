package booclient

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

type SQLArgs []interface{}

func (args SQLArgs) String() string {
	if len(args) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i := range args {
		if i > 0 {
			sb.WriteString(",")
		}
		if bs, ok := args[i].([]byte); ok {
			sb.WriteString("`")
			sb.Write(bs)
			sb.WriteString("`")
		} else if s, ok := args[i].(string); ok {
			sb.WriteString("`")
			sb.WriteString(s)
			sb.WriteString("`")
		} else if t, ok := args[i].(time.Time); ok {
			sb.WriteString("`")
			sb.WriteString(t.Format(time.RFC3339Nano))
			sb.WriteString("`")
		} else {
			fmt.Fprint(&sb, args[i])
		}
	}
	sb.WriteString("]")
	return sb.String()
}

func AnyArray(args []interface{}) fmt.Stringer {
	return SQLArgs(args)
}

func Error(err error) slog.Attr {
	return slog.Any("error", err)
}

// SQLTracer 是 github.com/runner-mei/GoBatis 的 Tracer
type SQLTracer struct {
	Logger *slog.Logger
	Level  slog.Level
}

func (w SQLTracer) Write(ctx context.Context, id, sql string, args []interface{}, err error) {
	if !w.Logger.Enabled(ctx, w.Level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, Infof]
	r := slog.NewRecord(time.Now(), w.Level, "execute sql '"+sql+"'", pcs[0])
	r.AddAttrs(slog.String("id", id),
		slog.Any("args", SQLArgs(args)),
		slog.Any("error", err))
	_ = w.Logger.Handler().Handle(ctx, r)
}

func NewSQLTracer(logger *slog.Logger, lvl ...slog.Level) SQLTracer {
	var level = slog.LevelInfo
	if len(lvl) > 0 {
		level = lvl[0]
	}
	return SQLTracer{
		Logger: logger,
		Level:  level,
	}
}
