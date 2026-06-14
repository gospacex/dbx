// Package config provides database configuration types.
package config

import "fmt"

// PoolConfig holds connection pool settings.
// Fields are defined in pool.go.
// TracingConfig holds tracing/export settings.
// Fields are defined in tracing.go.

// DBConfig is the unified interface for all 8 database config types.
type DBConfig interface {
	DriverName() string
	DSN() string
	Validate() error
	GetPool() *PoolConfig
	GetTracing() *TracingConfig
}

// CommonNetworkConfig holds network connection fields shared by 7/8 databases.
type CommonNetworkConfig struct {
	Host     string `yaml:"host" json:"host" toml:"host"`
	Port     int    `yaml:"port" json:"port" toml:"port"`
	Username string `yaml:"username" json:"username" toml:"username"`
	Password string `yaml:"password" json:"-" toml:"password"`
	Database string `yaml:"database" json:"database" toml:"database"`
}

// DSN builds a default MySQL-style DSN.
func (c *CommonNetworkConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.Username, c.Password, c.Host, c.Port, c.Database)
}

// BaseDBConfig embeds CommonNetworkConfig, PoolConfig, and TracingConfig.
type BaseDBConfig struct {
	CommonNetworkConfig `yaml:",inline" json:",inline" toml:",inline"`
	Pool     *PoolConfig    `yaml:"pool,omitempty" json:"pool,omitempty" toml:"pool,omitempty"`
	Tracing  *TracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty" toml:"tracing,omitempty"`
}

func (b *BaseDBConfig) GetPool() *PoolConfig      { return b.Pool }
func (b *BaseDBConfig) GetTracing() *TracingConfig { return b.Tracing }

// DSN builds a default MySQL-style DSN from CommonNetworkConfig.
func (b *BaseDBConfig) DSN() string {
	return b.CommonNetworkConfig.DSN()
}
