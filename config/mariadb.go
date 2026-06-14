package config

// MariaDBConfig implements DBConfig for MariaDB (wire-compatible with MySQL).
type MariaDBConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	Charset      string `yaml:"charset" json:"charset" toml:"charset"`
}

// DriverName returns the driver name.
func (c *MariaDBConfig) DriverName() string { return "mysql" }

// Validate validates the config, setting defaults.
func (c *MariaDBConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return errInvalidConfig("mariadb: host and username are required")
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

// DSN builds a MariaDB DSN string.
func (c *MariaDBConfig) DSN() string {
	return formatMySQLDSN(c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

var _ DBConfig = (*MariaDBConfig)(nil)
