// Package tidbx implements hubx.ClientProvider for "dbx.tidb".
package tidbx

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
func (p *Provider) Name() string { return "dbx.tidb" }

func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: dbx.tidb/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}
	var c Config
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml", ErrorUnused: true, Result: &c,
	})
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: dbx.tidb/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("%w: dbx.tidb/%s: dsn is required", hubx.ErrConfigInvalid, instanceName)
	}
	db, err := sql.Open("mysql", c.DSN)
	if err != nil {
		return nil, fmt.Errorf("%w: dbx.tidb/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	}
	return &client{db: db, driver: "tidb"}, nil
}

func (p *Provider) HealthCheck(context.Context) error { return nil }
func (p *Provider) Close() error                      { return nil }

type client struct {
	db     *sql.DB
	driver string
}

func (c *client) HealthCheck(ctx context.Context) error { return c.db.PingContext(ctx) }
func (c *client) Close() error                          { return c.db.Close() }
