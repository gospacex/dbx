package config

import "fmt"

// MySQLConfig implements DBConfig for MySQL.
type MySQLConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	Charset      string `yaml:"charset" json:"charset" toml:"charset"`
}

// DriverName returns the driver name.
func (c *MySQLConfig) DriverName() string { return "mysql" }

// Validate validates the config, setting defaults.
func (c *MySQLConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return fmt.Errorf("mysql: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 3306
	}
	if c.Charset == "" {
		c.Charset = "utf8mb4"
	}
	if c.Pool != nil {
		c.Pool.Validate()
	}
	if c.Tracing != nil {
		c.Tracing.Validate()
	}
	return nil
}

// DSN builds a MySQL DSN string.
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

var _ DBConfig = (*MySQLConfig)(nil)
