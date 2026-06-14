// Package dbsql provides GORM database access helpers.
package dbsql

import (
	"context"
	"fmt"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/orm"
	"gorm.io/gorm"
)

// Open opens a *gorm.DB from a DBConfig. If the config includes Tracing,
// the trace exporter is initialized before opening the connection.
func Open(cfg config.DBConfig) (*gorm.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("dbsql: %w", err)
	}
	if err := ExtractTracingAndApply(context.Background(), cfg.GetTracing()); err != nil {
		return nil, fmt.Errorf("dbsql: tracing: %w", err)
	}
	db, err := orm.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("dbsql: open: %w", err)
	}
	if p := cfg.GetPool(); p != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("dbsql: pool: %w", err)
		}
		sqlDB.SetMaxOpenConns(p.MaxOpenConns)
		sqlDB.SetMaxIdleConns(p.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(p.ConnMaxLifetimeDuration())
		sqlDB.SetConnMaxIdleTime(p.ConnMaxIdleTimeDuration())
	}
	return db, nil
}

// OpenPath loads a config from a file and opens a *gorm.DB.
// The file format (yaml/json/toml) is determined by the extension.
func OpenPath(path string) (*gorm.DB, error) {
	cfg, tc, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("dbsql: load: %w", err)
	}
	if tc != nil {
		if err := ExtractTracingAndApply(context.Background(), tc); err != nil {
			return nil, fmt.Errorf("dbsql: tracing: %w", err)
		}
	}
	db, err := orm.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("dbsql: open: %w", err)
	}
	if p := cfg.GetPool(); p != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("dbsql: pool: %w", err)
		}
		sqlDB.SetMaxOpenConns(p.MaxOpenConns)
		sqlDB.SetMaxIdleConns(p.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(p.ConnMaxLifetimeDuration())
		sqlDB.SetConnMaxIdleTime(p.ConnMaxIdleTimeDuration())
	}
	return db, nil
}
