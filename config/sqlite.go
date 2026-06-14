package config

import "fmt"

// SQLiteConfig implements DBConfig for SQLite (file-based, no network fields).
type SQLiteConfig struct {
	Path    string         `yaml:"path" json:"path" toml:"path"`
	Pool    *PoolConfig    `yaml:"pool,omitempty" json:"pool,omitempty" toml:"pool,omitempty"`
	Tracing *TracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty" toml:"tracing,omitempty"`
}

// DriverName returns the driver name.
func (c *SQLiteConfig) DriverName() string { return "sqlite" }

// DSN returns the file path.
func (c *SQLiteConfig) DSN() string { return c.Path }

// Validate validates the config.
func (c *SQLiteConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("sqlite: path is required")
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// GetPool returns the pool config.
func (c *SQLiteConfig) GetPool() *PoolConfig { return c.Pool }

// GetTracing returns the tracing config.
func (c *SQLiteConfig) GetTracing() *TracingConfig { return c.Tracing }

var _ DBConfig = (*SQLiteConfig)(nil)
