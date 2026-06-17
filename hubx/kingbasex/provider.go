// Package kingbasex implements hubx.ClientProvider for "dbx.kingbase".
package kingbasex

import (
	"context"
	"fmt"

	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	DSN             string `yaml:"dsn"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

type Provider struct{}

func New() *Provider             { return &Provider{} }
func (p *Provider) Name() string { return "dbx.kingbase" }

func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: dbx.kingbase/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}
	var c Config
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml", ErrorUnused: true, Result: &c,
	})
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: dbx.kingbase/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("%w: dbx.kingbase/%s: dsn is required", hubx.ErrConfigInvalid, instanceName)
	}
	return nil, fmt.Errorf("%w: dbx.kingbase/%s: driver not yet wired", hubx.ErrBuildFailed, instanceName)
}

func (p *Provider) HealthCheck(context.Context) error { return nil }
func (p *Provider) Close() error                      { return nil }

type client struct{}

func (c *client) HealthCheck(ctx context.Context) error { return nil }
func (c *client) Close() error                          { return nil }
