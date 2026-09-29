package orm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gospacex/logx"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type logEntry struct {
	level string
	msg   string
}

// fakeLogxLogger embeds logx.Logger (nil) so only the *Context methods the
// adapter actually calls need real implementations.
type fakeLogxLogger struct {
	logx.Logger
	entries []logEntry
}

func (f *fakeLogxLogger) InfoContext(_ context.Context, msg string, _ ...logx.Field) {
	f.entries = append(f.entries, logEntry{"info", msg})
}
func (f *fakeLogxLogger) WarnContext(_ context.Context, msg string, _ ...logx.Field) {
	f.entries = append(f.entries, logEntry{"warn", msg})
}
func (f *fakeLogxLogger) ErrorContext(_ context.Context, msg string, _ ...logx.Field) {
	f.entries = append(f.entries, logEntry{"error", msg})
}

func TestLogxLogger_TraceLevels(t *testing.T) {
	fake := &fakeLogxLogger{}
	l := NewLogxLogger(fake, LogxLoggerConfig{Level: "info"})
	ctx := context.Background()

	fc := func() (string, int64) { return "SELECT 1", 1 }

	l.Trace(ctx, time.Now(), fc, nil) // info path
	l.Trace(ctx, time.Now().Add(-time.Second), fc, nil)
	l.Trace(ctx, time.Now(), fc, errors.New("boom")) // error path

	if len(fake.entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(fake.entries))
	}
	if fake.entries[0].level != "info" || fake.entries[2].level != "error" {
		t.Errorf("levels = %v, want info … error", fake.entries)
	}
}

func TestLogxLogger_SlowThreshold(t *testing.T) {
	fake := &fakeLogxLogger{}
	l := NewLogxLogger(fake, LogxLoggerConfig{Level: "warn", SlowThreshold: time.Millisecond})
	fc := func() (string, int64) { return "SELECT pg_sleep(1)", 1 }

	l.Trace(context.Background(), time.Now().Add(-time.Second), fc, nil)
	if len(fake.entries) != 1 || fake.entries[0].level != "warn" {
		t.Fatalf("slow query should produce one warn, got %v", fake.entries)
	}

	l.Trace(context.Background(), time.Now(), fc, nil) // fast: below warn level
	if len(fake.entries) != 1 {
		t.Fatalf("fast query at warn level should be silent, got %v", fake.entries)
	}
}

func TestLogxLogger_IgnoreRecordNotFound(t *testing.T) {
	fake := &fakeLogxLogger{}
	l := NewLogxLogger(fake, LogxLoggerConfig{Level: "error", IgnoreRecordNotFound: true})
	fc := func() (string, int64) { return "SELECT ...", 0 }

	l.Trace(context.Background(), time.Now(), fc, gorm.ErrRecordNotFound)
	if len(fake.entries) != 0 {
		t.Fatalf("record-not-found should be ignored, got %v", fake.entries)
	}
}

func TestLogxLogger_SilentAndNilLogger(t *testing.T) {
	l := NewLogxLogger(nil, LogxLoggerConfig{Level: "silent"})
	l.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 1", 1 }, errors.New("x"))
	l.Info(context.Background(), "ignored")
	if l.LogMode(gormlogger.Info) == nil {
		t.Fatal("LogMode should return a logger")
	}
}

func TestWithLogxLogger_OverridesDefault(t *testing.T) {
	db, err := Open(&stubSource{}, WithLogxLogger(nil, LogxLoggerConfig{Level: "info"}))
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if db == nil {
		t.Fatal("db should not be nil")
	}
}

// stubSource is a minimal DialectorSource for sqlite in-memory.
type stubSource struct{}

func (stubSource) DriverName() string { return "sqlite" }
func (stubSource) DSN() string        { return ":memory:" }
