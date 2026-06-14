package config

import "fmt"

// PostgresConfig implements DBConfig for PostgreSQL.
type PostgresConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	SSLMode      string `yaml:"ssl_mode" json:"ssl_mode" toml:"ssl_mode"`
	TimeZone     string `yaml:"timezone" json:"timezone" toml:"timezone"`
	Schema       string `yaml:"schema" json:"schema" toml:"schema"`
}

// DriverName returns the driver name.
func (c *PostgresConfig) DriverName() string { return "postgres" }

// Validate validates the config and sets defaults.
func (c *PostgresConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return fmt.Errorf("postgres: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.TimeZone == "" {
		c.TimeZone = "UTC"
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// DSN builds a PostgreSQL connection string.
func (c *PostgresConfig) DSN() string {
	s := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode, c.TimeZone)
	if c.Schema != "" {
		s += " search_path=" + c.Schema
	}
	return s
}

var _ DBConfig = (*PostgresConfig)(nil)
