package orm

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// product is a minimal model used to exercise create/query/update/delete.
type product struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error: %v", err)
	}
	// AutoMigrate runs a CREATE TABLE; install the tracing callbacks
	// first so the migration span is also captured by tests that
	// assert span counts.
	installTracing(db)
	if err := db.AutoMigrate(&product{}); err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}
	return db
}

// withRecorder swaps the OTel global TracerProvider for one that
// records spans into the returned in-memory recorder, then restores
// the previous provider on test cleanup. Returns the recorder so the
// test can assert on captured spans.
func withRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	prev := otel.GetTracerProvider()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return rec
}

func TestInstallTracing_CreatesSpansForCRUD(t *testing.T) {
	db := newSQLiteDB(t)
	rec := withRecorder(t)

	ctx := context.Background()
	p := &product{Name: "alpha"}
	if err := db.WithContext(ctx).Create(p).Error; err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if err := db.WithContext(ctx).First(&product{}, p.ID).Error; err != nil {
		t.Fatalf("First error: %v", err)
	}
	if err := db.WithContext(ctx).Model(&product{}).Where("id = ?", p.ID).Update("name", "beta").Error; err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if err := db.WithContext(ctx).Delete(&product{}, p.ID).Error; err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	spans := rec.Ended()
	if got, want := len(spans), 4; got < want {
		t.Fatalf("got %d spans, want at least %d (create/query/update/delete)", got, want)
	}

	// Each GORM CRUD op should produce exactly one db.<op> span from
	// this instrumentation. Build a quick frequency map by name.
	got := map[string]int{}
	for _, s := range spans {
		got[s.Name()]++
	}
	for _, op := range []string{"db.create", "db.query", "db.update", "db.delete"} {
		if got[op] == 0 {
			t.Errorf("expected at least one span named %q, got names=%v", op, mapKeys(got))
		}
	}
}

func TestInstallTracing_AttributesOnEnd(t *testing.T) {
	db := newSQLiteDB(t)
	rec := withRecorder(t)

	p := &product{Name: "alpha"}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("Create error: %v", err)
	}

	spans := rec.Ended()
	var found bool
	for _, s := range spans {
		if s.Name() != "db.create" {
			continue
		}
		found = true
		// Build a set of attribute keys for stable assertions.
		keys := map[string]string{}
		for _, a := range s.Attributes() {
			keys[string(a.Key)] = a.Value.Emit()
		}
		if keys["db.system"] != "sqlite" {
			t.Errorf("db.system = %q, want sqlite", keys["db.system"])
		}
		if keys["db.operation"] != "create" {
			t.Errorf("db.operation = %q, want create", keys["db.operation"])
		}
		if keys["db.sql.table"] != "products" {
			t.Errorf("db.sql.table = %q, want products", keys["db.sql.table"])
		}
		if keys["db.query.text"] == "" {
			t.Error("db.query.text should be populated after the create")
		}
	}
	if !found {
		t.Fatal("no db.create span recorded")
	}
}

func TestInstallTracing_RecordNotFoundIsNotSpanError(t *testing.T) {
	db := newSQLiteDB(t)
	rec := withRecorder(t)

	// gorm.ErrRecordNotFound is treated as a soft miss.
	err := db.First(&product{}, 99999).Error
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}

	spans := rec.Ended()
	for _, s := range spans {
		if s.Name() != "db.query" {
			continue
		}
		if s.Status().Code.String() == "Error" {
			t.Errorf("db.query span should not be marked Error on ErrRecordNotFound, got status=%v", s.Status().Code.String())
		}
	}
}

func mapKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
