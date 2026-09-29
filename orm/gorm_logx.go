package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gospacex/logx"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// LogxLoggerConfig tunes the logx-backed GORM logger. The field names
// mirror the xmiddleware.sqlx.databases.<name> contract in configx's
// default config files (slow_threshold / log_level).
type LogxLoggerConfig struct {
	// Level is the minimum GORM log level: silent | error | warn | info.
	// Empty defaults to warn (same as orm.Open's built-in default).
	Level string
	// SlowThreshold flags queries slower than this as slow. <= 0 disables
	// slow-query detection.
	SlowThreshold time.Duration
	// IgnoreRecordNotFound downgrades gorm.ErrRecordNotFound from error
	// to nothing (it is a normal "miss", not a failure).
	IgnoreRecordNotFound bool
}

// logxLogger adapts a logx.Logger to gorm/logger.Interface. Log calls use
// the logx *Context variants, so when the statement context carries an
// active OTel span (installed by installTracing) the log entry is
// correlated with the call chain automatically.
type logxLogger struct {
	l              logx.Logger
	level          gormlogger.LogLevel
	slowThreshold  time.Duration
	ignoreNotFound bool
}

// NewLogxLogger returns a gorm/logger.Interface backed by l. A nil l
// falls back to logx.GetGlobal(), which is a safe no-op logger when no
// logx provider has been registered — so the adapter never panics.
func NewLogxLogger(l logx.Logger, cfg LogxLoggerConfig) gormlogger.Interface {
	if l == nil {
		l = logx.GetGlobal()
	}
	return &logxLogger{
		l:              l,
		level:          parseGormLogLevel(cfg.Level),
		slowThreshold:  cfg.SlowThreshold,
		ignoreNotFound: cfg.IgnoreRecordNotFound,
	}
}

func parseGormLogLevel(s string) gormlogger.LogLevel {
	switch s {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "info", "debug":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

// WithLogxLogger returns a gorm.Option that replaces the default GORM
// logger with a logx-backed one. Because orm.Open prepends its default
// *gorm.Config, this option is applied after it and wins.
func WithLogxLogger(l logx.Logger, cfg LogxLoggerConfig) gorm.Option {
	return &gorm.Config{Logger: NewLogxLogger(l, cfg)}
}

func (g *logxLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *g
	cp.level = level
	return &cp
}

func (g *logxLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if g.level >= gormlogger.Info {
		g.l.InfoContext(ctx, sprintf(msg, args...))
	}
}

func (g *logxLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if g.level >= gormlogger.Warn {
		g.l.WarnContext(ctx, sprintf(msg, args...))
	}
}

func (g *logxLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if g.level >= gormlogger.Error {
		g.l.ErrorContext(ctx, sprintf(msg, args...))
	}
}

func (g *logxLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []logx.Field{
		logx.String("db.elapsed", elapsed.String()),
		logx.Int("db.rows", int(rows)),
		logx.String("db.statement", sql),
	}
	switch {
	case err != nil && g.level >= gormlogger.Error:
		if g.ignoreNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		g.l.ErrorContext(ctx, "db trace error", append(fields, logx.Err(err))...)
	case g.slowThreshold > 0 && elapsed > g.slowThreshold && g.level >= gormlogger.Warn:
		g.l.WarnContext(ctx, "db slow query", fields...)
	case g.level >= gormlogger.Info:
		g.l.InfoContext(ctx, "db trace", fields...)
	}
}

// sprintf formats GORM's printf-style messages.
func sprintf(msg string, args ...interface{}) string {
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}
