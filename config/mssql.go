package config

import "fmt"

// MSSQLConfig implements DBConfig for Microsoft SQL Server.
type MSSQLConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	Instance     string `yaml:"instance" json:"instance" toml:"instance"`
}

// DriverName returns the driver name.
func (c *MSSQLConfig) DriverName() string { return "mssql" }

// Validate validates the config and sets defaults.
func (c *MSSQLConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return fmt.Errorf("mssql: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 1433
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// DSN builds a SQL Server connection string.
func (c *MSSQLConfig) DSN() string {
	if c.Instance != "" {
		return fmt.Sprintf("sqlserver://%s:%s@%s/%s?instance=%s", c.Username, c.Password, c.Host, c.Database, c.Instance)
	}
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d/%s", c.Username, c.Password, c.Host, c.Port, c.Database)
}

var _ DBConfig = (*MSSQLConfig)(nil)
