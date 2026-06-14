package config

import "fmt"

// OracleConfig implements DBConfig for Oracle Database.
type OracleConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	ServiceName  string `yaml:"service_name" json:"service_name" toml:"service_name"`
	SID          string `yaml:"sid" json:"sid" toml:"sid"`
}

// DriverName returns the driver name.
func (c *OracleConfig) DriverName() string { return "oracle" }

// Validate validates the config and sets defaults.
func (c *OracleConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return fmt.Errorf("oracle: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 1521
	}
	if c.ServiceName == "" && c.SID == "" {
		return fmt.Errorf("oracle: service_name or sid is required")
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// DSN builds an Oracle connection string.
func (c *OracleConfig) DSN() string {
	if c.ServiceName != "" {
		return fmt.Sprintf("oracle://%s:%s@%s:%d/%s", c.Username, c.Password, c.Host, c.Port, c.ServiceName)
	}
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s?sid=%s", c.Username, c.Password, c.Host, c.Port, c.Database, c.SID)
}

var _ DBConfig = (*OracleConfig)(nil)
