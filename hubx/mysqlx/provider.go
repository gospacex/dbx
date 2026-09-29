// Package mysqlx implements hubx.ClientProvider for "dbx.mysql".
package mysqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gospacex/dbx/orm"
	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
)

type Config struct {
	// Dsn 必须 exported — mapstructure 通过反射写入,unexported 字段会触发
	// ErrorUnused("'' has invalid keys: dsn") 而非无声跳过,导致 build 永远失败。
	// 但 field 名不能与方法名 DSN() 同名,因此用 Dsn(大写但非全大写)。
	Dsn             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// DriverName implements orm.DialectorSource.
func (c *Config) DriverName() string { return "mysql" }

// DSN implements orm.DialectorSource.
func (c *Config) DSN() string { return c.Dsn }

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
	// StringToTimeDurationHookFunc lets `conn_max_lifetime: "30m"` decode.
	// Without it a duration string is a hard decode error, and a bare
	// integer silently lands as nanoseconds (1800 -> 1.8us), which expires
	// pooled connections almost immediately. See README §8.
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnused: true,
		DecodeHook:  mapstructure.StringToTimeDurationHookFunc(),
		Result:      &c,
	})
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if c.Dsn == "" {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: dsn is required", hubx.ErrConfigInvalid, instanceName)
	}
	gormDB, err := orm.Open(&c)
	if err != nil {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	// Apply pool settings via *sql.DB underlying connection.
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("%w: dbx.mysql/%s: failed to get underlying sql.DB: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	if c.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	}
	return &client{gormDB: gormDB, sqlDB: sqlDB}, nil
}

func (p *Provider) HealthCheck(context.Context) error { return nil }
func (p *Provider) Close() error                      { return nil }

type client struct {
	gormDB *gorm.DB
	sqlDB  *sql.DB
}

func (c *client) HealthCheck(ctx context.Context) error { return c.sqlDB.PingContext(ctx) }
func (c *client) Close() error                          { return c.sqlDB.Close() }

// DB returns the underlying *sql.DB for raw SQL operations.
func (c *client) DB() *sql.DB { return c.sqlDB }

// GORM returns the *gorm.DB for GORM-based operations.
func (c *client) GORM() *gorm.DB { return c.gormDB }
