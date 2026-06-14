package config

import "fmt"

// GaussDBConfig implements DBConfig for Huawei GaussDB (wire-compatible with PostgreSQL).
type GaussDBConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	SSLMode      string `yaml:"ssl_mode" json:"ssl_mode" toml:"ssl_mode"`
	Schema       string `yaml:"schema" json:"schema" toml:"schema"`
}

// DriverName returns the driver name.
func (c *GaussDBConfig) DriverName() string { return "postgres" }

// Validate validates the config and sets defaults.
func (c *GaussDBConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return fmt.Errorf("gaussdb: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// DSN builds a GaussDB DSN string (PostgreSQL wire-compatible, no TimeZone).
func (c *GaussDBConfig) DSN() string {
	s := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
	if c.Schema != "" {
		s += " search_path=" + c.Schema
	}
	return s
}

var _ DBConfig = (*GaussDBConfig)(nil)
