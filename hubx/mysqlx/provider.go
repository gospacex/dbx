// Package mysqlx implements hubx.ClientProvider for "dbx.mysql".
package mysqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type Provider struct{}

func New() *Provider             { return &Provider{} }
func (p *Provider) Name() string { return "dbx.mysql" }

func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}
	var c Config
	// ErrorUnused catches typos in the YAML (e.g. `dsnn` instead of `dsn`).
	// ErrorUnset is intentionally OFF: max_open_conns / max_idle_conns /
	// conn_max_lifetime are tuning knobs with sensible zero defaults, so
	// they must not be required to be present. Only dsn is required, and
	// that's enforced explicitly below.
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml", ErrorUnused: true, Result: &c,
	})
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: dsn is required", hubx.ErrConfigInvalid, instanceName)
	}
	db, err := sql.Open("mysql", c.DSN)
	if err != nil {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	}
	return &client{db: db, driver: "mysql"}, nil
}

func (p *Provider) HealthCheck(context.Context) error { return nil }
func (p *Provider) Close() error                      { return nil }

type client struct {
	db     *sql.DB
	driver string
}

func (c *client) HealthCheck(ctx context.Context) error { return c.db.PingContext(ctx) }
func (c *client) Close() error                          { return c.db.Close() }
