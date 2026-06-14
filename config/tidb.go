package config

// TiDBConfig implements DBConfig for TiDB (wire-compatible with MySQL).
type TiDBConfig struct {
	BaseDBConfig `yaml:",inline" json:",inline" toml:",inline"`
	Charset      string `yaml:"charset" json:"charset" toml:"charset"`
}

// DriverName returns the driver name.
func (c *TiDBConfig) DriverName() string { return "mysql" }

// Validate validates the config, setting defaults.
func (c *TiDBConfig) Validate() error {
	if c.Host == "" || c.Username == "" {
		return errInvalidConfig("tidb: host and username are required")
	}
	if c.Port == 0 {
		c.Port = 4000
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

// DSN builds a TiDB DSN string.
func (c *TiDBConfig) DSN() string {
	return formatMySQLDSN(c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

var _ DBConfig = (*TiDBConfig)(nil)

// shared helpers for MySQL-compatible DSN building

func formatMySQLDSN(user, pass, host string, port int, db, charset string) string {
	return user + ":" + pass + "@tcp(" + host + ":" + itoa(port) + ")/" + db + "?charset=" + charset + "&parseTime=True&loc=Local"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func errInvalidConfig(msg string) error {
	return &invalidConfigError{msg: msg}
}

type invalidConfigError struct {
	msg string
}

func (e *invalidConfigError) Error() string { return e.msg }
