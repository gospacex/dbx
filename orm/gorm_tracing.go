package orm

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gorm.io/gorm"
)

// GORM v2 does NOT auto-pick up otel.GetTracerProvider(): its built-in
// tracing system is opt-in via a plugin or hand-written callbacks. dbx
// ships its own minimal callback registration so the 4 SQL-issuing
// callback groups (create / query / update / delete) emit spans to
// whichever TracerProvider the caller has registered as the OTel
// global — without taking on an extra dependency.
//
// The auto-migrate + 5 CRUD steps in the example (Create, Get, List,
// Update, Delete) all flow through these 4 groups, so spans appear
// in Jaeger / Kafka / Redis Stream automatically once the caller
// calls dbsql.CreateExporter and otel.SetTracerProvider.

const (
	// ormTracerName identifies this instrumentation in OTel backends.
	ormTracerName = "github.com/gospacex/dbx/orm"
	// ormSpanKey is the sync.Map key used to pass a span from the
	// before-callback to the after-callback via Statement.Settings.
	ormSpanKey = "dbx.orm.span"
)

// installTracing registers OTel tracing callbacks on db for the 4
// SQL-issuing GORM v2 callback groups. Spans are routed through the
// OTel global TracerProvider, so they work with all 3 dbx trace
// exporters (jaeger, kafka, redis_stream) as long as the caller has
// set up a TracerProvider first.
//
// When no TracerProvider has been registered, otel.Tracer returns a
// no-op tracer and the callbacks are free — so installTracing is
// always safe to call.
//
// Idempotent: GORM's callback.Register replaces a callback when the
// name already exists, so calling installTracing twice on the same
// *gorm.DB is harmless.
func installTracing(db *gorm.DB) {
	for _, op := range []string{"create", "query", "update", "delete"} {
		before, after := tracingBeforeAfter(op)
		cbs := db.Callback()
		switch op {
		case "create":
			cbs.Create().Before("gorm:before_create").Register("dbx:trace_before_create", before)
			cbs.Create().After("gorm:after_create").Register("dbx:trace_after_create", after)
		case "query":
			cbs.Query().Before("gorm:before_query").Register("dbx:trace_before_query", before)
			cbs.Query().After("gorm:after_query").Register("dbx:trace_after_query", after)
		case "update":
			cbs.Update().Before("gorm:before_update").Register("dbx:trace_before_update", before)
			cbs.Update().After("gorm:after_update").Register("dbx:trace_after_update", after)
		case "delete":
			cbs.Delete().Before("gorm:before_delete").Register("dbx:trace_before_delete", before)
			cbs.Delete().After("gorm:after_delete").Register("dbx:trace_after_delete", after)
		}
	}
}

// tracingBeforeAfter returns the (before, after) pair of GORM callbacks
// for the given operation. The before-callback starts an OTel span and
// stores it on the Statement; the after-callback retrieves it, attaches
// final attributes (db.system, db.sql.table, db.query.text), records
// any non-soft error, and ends the span.
func tracingBeforeAfter(op string) (before, after func(*gorm.DB)) {
	before = func(db *gorm.DB) {
		if db == nil || db.Statement == nil {
			return
		}
		ctx := db.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}
		tracer := otel.Tracer(ormTracerName)
		ctx, span := tracer.Start(ctx, "db."+op,
			trace.WithSpanKind(trace.SpanKindClient),
		)
		db.Statement.Settings.Store(ormSpanKey, span)
		db.Statement.Context = ctx
	}
	after = func(db *gorm.DB) {
		if db == nil || db.Statement == nil {
			return
		}
		v, ok := db.Statement.Settings.LoadAndDelete(ormSpanKey)
		if !ok {
			return
		}
		span, ok := v.(trace.Span)
		if !ok {
			return
		}
		attrs := make([]attribute.KeyValue, 0, 4)
		attrs = append(attrs, attribute.String("db.operation", op))
		if sys := dbSystemFromDialector(db.Dialector.Name()); sys.Key != "" {
			attrs = append(attrs, sys)
		}
		if tbl := db.Statement.Table; tbl != "" {
			attrs = append(attrs, attribute.String("db.sql.table", tbl))
		}
		if sqlStr := db.Statement.SQL.String(); sqlStr != "" {
			attrs = append(attrs, attribute.String("db.query.text", sqlStr))
		}
		span.SetAttributes(attrs...)

		// gorm.ErrRecordNotFound is a soft "not found" miss; not a
		// span error. Anything else is.
		if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
			span.RecordError(db.Error)
			span.SetStatus(codes.Error, db.Error.Error())
		}
		span.End()
	}
	return before, after
}

// dbSystemFromDialector maps a GORM dialector name to the OTel
// semconv db.system attribute. Returns the zero KeyValue when the
// driver is unknown so the caller can detect the miss with `sys.Key == ""`.
func dbSystemFromDialector(name string) attribute.KeyValue {
	switch name {
	case "mysql", "tidb", "mariadb":
		return attribute.String("db.system", "mysql")
	case "postgres", "gaussdb":
		return attribute.String("db.system", "postgresql")
	case "mssql":
		return attribute.String("db.system", "microsoft.sql_server")
	case "sqlite":
		return attribute.String("db.system", "sqlite")
	case "oracle":
		return attribute.String("db.system", "oracle")
	}
	return attribute.KeyValue{}
}
