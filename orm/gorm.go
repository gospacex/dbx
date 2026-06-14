// Package orm provides GORM dialector dispatch for 8 database drivers.
package orm

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DialectorSource is the minimal interface Open needs from a config: the
// driver name and the connection string. Every *XxxConfig in the config
// package already has both methods, so accepting a narrower interface here
// lets callers that don't need Validate/GetPool/GetTracing (e.g. a
// read/write-split cluster source) avoid implementing empty stubs.
type DialectorSource interface {
	DriverName() string
	DSN() string
}

// Open opens a *gorm.DB from a DialectorSource. Accepting a narrow interface
// keeps callers like dbsql.Open (which needs the full DBConfig) decoupled
// from callers like dbsql.OpenCluster (which only needs driver+DSN).
func Open(cfg DialectorSource, opts ...gorm.Option) (*gorm.DB, error) {
	d, err := Dialector(cfg.DriverName(), cfg.DSN())
	if err != nil {
		return nil, err
	}
	gormOpts := append([]gorm.Option{&gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}}, opts...)
	db, err := gorm.Open(d, gormOpts...)
	if err != nil {
		return nil, err
	}
	// installTracing registers OTel callbacks for create/query/update/
	// delete so every SQL operation emits a span to the OTel global
	// TracerProvider. No-op when no provider is registered, so it is
	// always safe to call.
	installTracing(db)
	return db, nil
}

// Dialector returns a gorm.Dialector for the given driver name and DSN.
// The 8 supported names are: mysql, postgres, tidb, mariadb, gaussdb, mssql, oracle, sqlite.
// TiDB and MariaDB wire-compatibly use the mysql dialector; GaussDB uses the postgres dialector.
func Dialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "mysql", "tidb", "mariadb":
		return mysql.New(mysql.Config{DSN: dsn}), nil
	case "postgres", "gaussdb":
		return postgres.New(postgres.Config{DSN: dsn}), nil
	case "mssql":
		return sqlserver.New(sqlserver.Config{DSN: dsn}), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	case "oracle":
		return nil, fmt.Errorf("gorm: oracle driver requires CGo (godror) or pure-Go driver")
	default:
		return nil, fmt.Errorf("gorm: unsupported driver %q", driver)
	}
}

// SupportedDrivers returns the list of supported driver names.
func SupportedDrivers() []string {
	return []string{"mysql", "postgres", "tidb", "mariadb", "gaussdb", "mssql", "oracle", "sqlite"}
}
