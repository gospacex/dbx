package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/gorm/logger"
)

// consoleLogger implements gorm/logger.Interface and writes to the
// standard log package. It supports:
//   - level filtering (silent / error / warn / info)
//   - slow query flagging via slowThreshold
//   - ANSI color toggle (auto-disabled when NO_COLOR=1)
//   - short-circuiting SQL output entirely when level=silent
type consoleLogger struct {
	level                  logger.LogLevel
	color                  bool
	slowThreshold          time.Duration
	ignoreRecordNotFound   bool
}

// newConsoleLogger builds a consoleLogger from the yaml-shaped config.
// Defaults match logger.Info when callers do not specify a level.
func newConsoleLogger(cfg loggerConfig) *consoleLogger {
	color := cfg.Color
	if os.Getenv("NO_COLOR") != "" {
		color = false
	}
	lvl := parseLogLevel(cfg.Level)
	return &consoleLogger{
		level:                lvl,
		color:                color,
		slowThreshold:        time.Duration(cfg.SlowThresholdMS) * time.Millisecond,
		ignoreRecordNotFound: cfg.IgnoreRecordNotFound,
	}
}

// loggerConfig is the yaml-friendly shape mirrored from mysql.example.yaml.
type loggerConfig struct {
	Level                string `yaml:"level" json:"level"`
	Color                bool   `yaml:"color" json:"color"`
	SlowThresholdMS      int    `yaml:"slow_threshold_ms" json:"slow_threshold_ms"`
	IgnoreRecordNotFound bool   `yaml:"ignore_record_not_found" json:"ignore_record_not_found"`
}

func parseLogLevel(s string) logger.LogLevel {
	switch s {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info", "":
		return logger.Info
	}
	return logger.Info
}

// LogMode returns a copy with the level adjusted. GORM calls this to
// switch verbosity per session; the underlying gormDB stores the
// returned Interface so the copy must be safe to use independently.
func (l *consoleLogger) LogMode(level logger.LogLevel) logger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

// Info prints at the info level.
func (l *consoleLogger) Info(_ context.Context, msg string, data ...interface{}) {
	if l.level < logger.Info {
		return
	}
	log.Printf(colorize("[INFO] ", "\033[36m", l.color)+msg, data...)
}

// Warn prints at the warn level.
func (l *consoleLogger) Warn(_ context.Context, msg string, data ...interface{}) {
	if l.level < logger.Warn {
		return
	}
	log.Printf(colorize("[WARN] ", "\033[33m", l.color)+msg, data...)
}

// Error prints at the error level.
func (l *consoleLogger) Error(_ context.Context, msg string, data ...interface{}) {
	if l.level < logger.Error {
		return
	}
	log.Printf(colorize("[ERROR] ", "\033[31m", l.color)+msg, data...)
}

// Trace prints the SQL statement and timing. When the elapsed time
// exceeds slowThreshold, a warning marker is appended. When level is
// silent, Trace is a no-op so SQL output is fully suppressed.
func (l *consoleLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && (!l.ignoreRecordNotFound || !isRecordNotFound(err)):
		log.Printf(colorize("[SQL] ", "\033[31m", l.color)+"%s | rows=%d | took=%.3fms | err=%v",
			sql, rows, float64(elapsed.Microseconds())/1000.0, err)
	case elapsed > l.slowThreshold && l.slowThreshold > 0:
		log.Printf(colorize("[SQL] ", "\033[33m", l.color)+"%s | rows=%d | took=%.3fms | ⚠️ slow query > %dms",
			sql, rows, float64(elapsed.Microseconds())/1000.0, l.slowThreshold.Milliseconds())
	default:
		log.Printf(colorize("[SQL] ", "\033[32m", l.color)+"%s | rows=%d | took=%.3fms",
			sql, rows, float64(elapsed.Microseconds())/1000.0)
	}
}

// colorize wraps s in the given ANSI escape when color is enabled.
// Kept as a free function so each log site stays a one-liner.
func colorize(s, code string, color bool) string {
	if !color {
		return s
	}
	return fmt.Sprintf("\033[1m%s\033[0m%s", code, s)
}
