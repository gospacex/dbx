package main

import (
	"context"
	"fmt"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// hardcodedMySQLConfig and hardcodedClusterConfig are the
// representative configs used by the `o` and `oc` subcommands. They
// are kept here (not in main.go) so the wiring in db.go is the single
// place that knows how to translate a hardcoded struct into a *gorm.DB.
//
// Password defaults to "REPLACE_ME"; main.go intercepts this and exits
// with code 2 before the dbsql.Open call.
var hardcodedMySQLConfig = &config.MySQLConfig{
	BaseDBConfig: config.BaseDBConfig{
		CommonNetworkConfig: config.CommonNetworkConfig{
			Host:     "127.0.0.1",
			Port:     3306,
			Username: "root",
			Password: "REPLACE_ME",
			Database: "example",
		},
		Pool: &config.PoolConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: 1800,
			ConnMaxIdleTime: 600,
		},
	},
	Charset: "utf8mb4",
}

var hardcodedClusterConfig = &config.ClusterConfig{
	Driver: "mysql",
	Sources: []config.NodeConfig{
		{
			Host:     "127.0.0.1",
			Port:     3306,
			Username: "root",
			Password: "REPLACE_ME",
			Database: "example",
		},
	},
	Replicas: []config.NodeConfig{
		{
			Host:     "127.0.0.1",
			Port:     3307,
			Username: "root",
			Password: "REPLACE_ME",
			Database: "example",
		},
	},
	LoadBalance: config.LoadBalanceRoundRobin,
	Pool: &config.PoolConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 1800,
		ConnMaxIdleTime: 600,
	},
}

// openSingleFromStruct opens a *gorm.DB from the hardcoded MySQL config.
// Maps to `dbsql.Open`. Returns a shutdown closure that callers MUST defer.
func openSingleFromStruct(ctx context.Context) (*gorm.DB, func(context.Context) error, error) {
	tc := hardcodedMySQLConfig.GetTracing()
	shutdownTP, err := setupTracerProvider(ctx, tc)
	if err != nil {
		return nil, nil, fmt.Errorf("open: tracing: %w", err)
	}

	db, err := dbsql.Open(hardcodedMySQLConfig)
	if err != nil {
		_ = shutdownTP(ctx)
		return nil, nil, fmt.Errorf("open: dbsql.Open: %w", err)
	}
	return db, buildShutdown(db, shutdownTP), nil
}

// openSingleFromYAML opens a *gorm.DB from a yaml file. Maps to
// `dbsql.OpenPath`. The yaml schema is the mqx-style `mysql:` /
// `pool:` / `trace:` / `logger:`; this helper unwraps the trace
// section, translates it to dbx internal schema, and re-stitches
// it onto the config before opening.
//
// The shutdown order is preserved via the closure built by buildShutdown.
func openSingleFromYAML(ctx context.Context, path string) (*gorm.DB, func(context.Context) error, error) {
	dbCfg, mqTrace, err := loadSingleFromYAML(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}
	tc := mqxTraceToDBX(mqTrace)
	if dbCfgMySQL, ok := dbCfg.(*config.MySQLConfig); ok {
		dbCfgMySQL.Tracing = tc
	}

	shutdownTP, err := setupTracerProvider(ctx, tc)
	if err != nil {
		return nil, nil, fmt.Errorf("open: tracing: %w", err)
	}

	db, err := dbsql.Open(dbCfg)
	if err != nil {
		_ = shutdownTP(ctx)
		return nil, nil, fmt.Errorf("open: dbsql.Open: %w", err)
	}
	return db, buildShutdown(db, shutdownTP), nil
}

// openClusterFromStruct opens a *gorm.DB from the hardcoded cluster
// config. Maps to `dbsql.OpenCluster`. The trace config on the
// cluster itself drives exporter setup.
func openClusterFromStruct(ctx context.Context) (*gorm.DB, func(context.Context) error, error) {
	tc := hardcodedClusterConfig.Tracing
	shutdownTP, err := setupTracerProvider(ctx, tc)
	if err != nil {
		return nil, nil, fmt.Errorf("open: tracing: %w", err)
	}

	db, err := dbsql.OpenCluster(hardcodedClusterConfig)
	if err != nil {
		_ = shutdownTP(ctx)
		return nil, nil, fmt.Errorf("open: dbsql.OpenCluster: %w", err)
	}
	return db, buildShutdown(db, shutdownTP), nil
}

// openClusterFromYAML opens a *gorm.DB from a cluster yaml file.
// Maps to `dbsql.OpenClusterPath`.
func openClusterFromYAML(ctx context.Context, path string) (*gorm.DB, func(context.Context) error, error) {
	cc, mqTrace, err := loadClusterFromYAML(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}
	cc.Tracing = mqxTraceToDBX(mqTrace)

	shutdownTP, err := setupTracerProvider(ctx, cc.Tracing)
	if err != nil {
		return nil, nil, fmt.Errorf("open: tracing: %w", err)
	}

	db, err := dbsql.OpenCluster(cc)
	if err != nil {
		_ = shutdownTP(ctx)
		return nil, nil, fmt.Errorf("open: dbsql.OpenCluster: %w", err)
	}
	return db, buildShutdown(db, shutdownTP), nil
}

// buildShutdown composes the 3-step shutdown sequence:
//   1. TracerProvider.Shutdown  (stops accepting new spans)
//   2. TracerProvider.ForceFlush (drains the batcher)
//   3. sqlDB.Close               (returns connections to the pool)
//
// The order is fixed and matches tasks.md §6 acceptance criterion 9.19.
func buildShutdown(db *gorm.DB, shutdownTP func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		// Step 1: flush tracing first so the span exporter gets
		// the chance to deliver pending telemetry while the DB
		// connection is still open.
		if err := shutdownTP(ctx); err != nil {
			return err
		}
		// Step 2: close the underlying *sql.DB. This is the
		// final step because gorm/logger traces through
		// sqlDB on Close, so the span exporter must already be
		// set up to receive those final spans.
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("shutdown: get sqlDB: %w", err)
		}
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("shutdown: sqlDB.Close: %w", err)
		}
		return nil
	}
}

// asGormLogger wires the consoleLogger into GORM via a Logger option.
// Used when callers want SQL output at the gorm level.
func asGormLogger(l *consoleLogger) logger.Interface {
	if l == nil {
		return logger.Default
	}
	return l
}
