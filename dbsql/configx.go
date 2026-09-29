package dbsql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/orm"
	"github.com/gospacex/logx"
	"github.com/gospacex/traceflag"
	"gorm.io/gorm"
)

// SQLConfig mirrors the xmiddleware.sqlx.databases.<name> section in
// configx's default config files (configx/configs/config.enterprise.yaml,
// config.simple.yaml). Field semantics follow that contract exactly:
// durations are time.ParseDuration strings ("5m"), not seconds.
type SQLConfig struct {
	// Driver is the ORM driver from xmiddleware.sqlx.driver. Only "gorm"
	// is implemented; "xorm" and "sqlx" are reserved (see FEATURE_MATRIX.md).
	Driver string
	// Dialect is the database dialect (mysql/postgres/sqlite/mssql/...).
	// Optional in the contract; when empty it is inferred from the DSN.
	Dialect string
	// Dsn is the full driver-specific connection string. Named Dsn (not
	// DSN) so the type can implement orm.DialectorSource's DSN() method —
	// same convention as the hubx providers.
	Dsn string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// SlowThreshold flags queries slower than this in the log. 0 disables.
	SlowThreshold time.Duration
	// EnableLog wires the logx-backed GORM logger (db.statement field
	// carries the executed SQL, correlated with the call chain).
	EnableLog bool
	// LogLevel: silent | error | warn | info. Empty defaults to warn.
	LogLevel string
	// EnableTrace wires the OTel callbacks so SQL statements enter the
	// call chain as span attribute db.query.text. When true,
	// OpenFromConfigx also provisions a TracerProvider from Tracing, so the
	// switch is not a no-op: without a provider every span is dropped by the
	// noop tracer that otel installs by default.
	//
	// Ordering is decided by ExtractTracingAndApply's first-init-wins rule,
	// which is scoped to that function: a TracerProvider dbx created earlier
	// is kept and Tracing is ignored, but a provider the host application
	// installed directly via otel.SetTracerProvider is NOT detected, and dbx
	// will replace it. Applications that own the global provider should
	// therefore leave enable_trace off and install the callbacks themselves.
	EnableTrace bool
	// Tracing carries the xmiddleware.sqlx.databases.<name>.tracing
	// sub-section (exporter/endpoint/sampler/...). Nil when neither
	// tracing nor enable_trace is configured. Read-only for callers; the
	// defaulting of exporter/endpoint happens in OpenFromConfigx via
	// TracingConfig.Validate, so ReadConfigFromConfigx stays non-failing.
	Tracing *config.TracingConfig

	// HealthCheck, when enabled, is served by (*gorm.DB).DB().PingContext;
	// dbx does not run a background pinger (see FEATURE_MATRIX.md).
	HealthCheck HealthCheckConfig
}

// HealthCheckConfig mirrors the health_check sub-section of the contract.
type HealthCheckConfig struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
}

// ReadConfigFromConfigx reads dbx config from a configx handle.
// Read path: xmiddleware.sqlx.databases.<name>; empty name means "default".
// Missing keys keep their zero values (duration 0 = no limit, matching
// database/sql semantics), so the function never fails.
func ReadConfigFromConfigx(cfg configx.Config, name string) *SQLConfig {
	if name == "" {
		name = "default"
	}
	prefix := fmt.Sprintf("xmiddleware.sqlx.databases.%s", name)

	c := &SQLConfig{
		Driver:          cfg.GetString("xmiddleware.sqlx.driver"),
		Dialect:         cfg.GetString(prefix + ".dialect"),
		Dsn:             cfg.GetString(prefix + ".dsn"),
		MaxOpenConns:    cfg.GetInt(prefix + ".max_open_conns"),
		MaxIdleConns:    cfg.GetInt(prefix + ".max_idle_conns"),
		ConnMaxLifetime: cfg.GetDuration(prefix + ".conn_max_lifetime"),
		ConnMaxIdleTime: cfg.GetDuration(prefix + ".conn_max_idle_time"),
		SlowThreshold:   cfg.GetDuration(prefix + ".slow_threshold"),
		EnableLog:       cfg.GetBool(prefix + ".enable_log"),
		LogLevel:        cfg.GetString(prefix + ".log_level"),
		EnableTrace:     cfg.GetBool(prefix + ".enable_trace"),
	}
	c.HealthCheck.Enabled = cfg.GetBool(prefix + ".health_check.enabled")
	c.HealthCheck.Interval = cfg.GetDuration(prefix + ".health_check.interval")
	c.HealthCheck.Timeout = cfg.GetDuration(prefix + ".health_check.timeout")
	// The tracing sub-section wins over the legacy enable_trace boolean,
	// matching config.TracingFromConfigx. Deriving EnableTrace from the
	// merged result keeps "install callbacks" and "have a provider" in
	// agreement, so enable_trace: true alone is enough to trace.
	c.Tracing = config.TracingFromConfigx(cfg, prefix)
	c.EnableTrace = c.Tracing != nil && c.Tracing.Enabled
	// 经统一开关收敛：全局总闸关闭时，GORM 的 trace callback 也不安装，
	// 避免"装了 callback 但 Provider 是 noop"的空转状态。
	c.EnableTrace = traceflag.Apply("dbx", c.EnableTrace)
	return c
}

// DriverName implements orm.DialectorSource.
func (c *SQLConfig) DriverName() string { return c.Dialect }

// DSN implements orm.DialectorSource.
func (c *SQLConfig) DSN() string { return c.Dsn }

// detectDialect infers the database dialect from a DSN string when the
// config does not name one explicitly. A URL-shaped DSN whose scheme is not
// recognized is rejected rather than silently treated as sqlite: a
// "postgres://user@host/db" style DSN used to fall through to the sqlite
// branch and surface as a confusing "unable to open database file".
func detectDialect(dsn string) (string, error) {
	switch {
	case strings.Contains(dsn, "@tcp("), strings.Contains(dsn, "parseTime"):
		return "mysql", nil
	case strings.HasPrefix(dsn, "host="), strings.Contains(dsn, " sslmode="),
		strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return "postgres", nil
	case strings.HasPrefix(dsn, "sqlserver://"):
		return "mssql", nil
	case strings.Contains(dsn, "://"):
		// Only the scheme is echoed: a DSN carries a password.
		return "", fmt.Errorf("dbsql: cannot infer dialect from dsn with scheme %q, set dialect explicitly", dsnScheme(dsn))
	default:
		// Scheme-less strings are sqlite paths: ":memory:", "file:...",
		// "/tmp/x.db", "app.db" and bare filenames.
		return "sqlite", nil
	}
}

// dsnScheme returns the "<scheme>://" prefix of dsn, or "" when dsn is not
// URL-shaped. Keeping it separate from detectDialect lets error messages
// name the offending scheme without echoing credentials.
func dsnScheme(dsn string) string {
	if i := strings.Index(dsn, "://"); i > 0 {
		return dsn[:i+3]
	}
	return ""
}

// OpenFromConfigx opens a *gorm.DB from a configx handle using the
// xmiddleware.sqlx contract. l may be nil — the logx adapter then falls
// back to logx.GetGlobal() (a safe no-op when logx has no provider).
//
// When EnableTrace is on, the tracing sub-section (or, absent it, the
// jaeger defaults from TracingConfig.Validate) is applied to the global OTel
// TracerProvider before the connection is opened, so the SQL statements
// issued later do enter the call chain. ExtractTracingAndApply is
// first-init-wins, so a provider installed earlier by the caller wins.
//
// Callers MUST defer ShutdownTracerProvider(ctx) before application exit
// to flush pending spans and release resources when tracing is enabled.
func OpenFromConfigx(cfg configx.Config, name string, l logx.Logger) (*gorm.DB, error) {
	sc := ReadConfigFromConfigx(cfg, name)
	if sc.Dsn == "" {
		return nil, fmt.Errorf("dbsql: xmiddleware.sqlx.databases.%s.dsn is required", orDefault(name))
	}
	if sc.Driver != "" && sc.Driver != "gorm" {
		return nil, fmt.Errorf("dbsql: orm driver %q is reserved but not implemented (supported: gorm)", sc.Driver)
	}
	if sc.Dialect == "" {
		dialect, err := detectDialect(sc.Dsn)
		if err != nil {
			return nil, err
		}
		sc.Dialect = dialect
	}

	if sc.EnableTrace {
		if err := applyTracing(sc.Tracing); err != nil {
			return nil, err
		}
	}

	var opts []gorm.Option
	if sc.EnableLog {
		opts = append(opts, orm.WithLogxLogger(l, orm.LogxLoggerConfig{
			Level:                sc.LogLevel,
			SlowThreshold:        sc.SlowThreshold,
			IgnoreRecordNotFound: true,
		}))
	}

	openFn := orm.Open
	if !sc.EnableTrace {
		openFn = orm.OpenNoTrace
	}
	db, err := openFn(sc, opts...)
	if err != nil {
		return nil, fmt.Errorf("dbsql: configx open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("dbsql: configx pool: %w", err)
	}
	// Zero values are meaningful here (database/sql: 0 = unlimited), so
	// apply them unconditionally — the contract uses durations, and an
	// absent key means "no limit", not "dbx default".
	sqlDB.SetMaxOpenConns(sc.MaxOpenConns)
	sqlDB.SetMaxIdleConns(sc.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(sc.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(sc.ConnMaxIdleTime)
	return db, nil
}

func orDefault(name string) string {
	if name == "" {
		return "default"
	}
	return name
}

// applyTracing provisions the global TracerProvider described by the
// xmiddleware.sqlx.tracing sub-section. A nil tc means the config only set
// enable_trace: true, in which case the defaults from
// TracingConfig.Validate (jaeger over http/protobuf to localhost:4318) apply
// — the same fallback the file-config path uses, so the switch is never a
// no-op. Validate must run first: it both rejects unknown exporters and
// fills the fields CreateExporter dereferences.
func applyTracing(tc *config.TracingConfig) error {
	if tc == nil {
		tc = &config.TracingConfig{Enabled: true}
	}
	if err := tc.Validate(); err != nil {
		return fmt.Errorf("dbsql: configx tracing: %w", err)
	}
	if err := ExtractTracingAndApply(context.Background(), tc); err != nil {
		return fmt.Errorf("dbsql: configx tracing: %w", err)
	}
	return nil
}
