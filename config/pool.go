package config

import "time"

// PoolConfig holds database connection pool settings.
type PoolConfig struct {
	MaxOpenConns    int `yaml:"max_open_conns" json:"max_open_conns" toml:"max_open_conns"`
	MaxIdleConns    int `yaml:"max_idle_conns" json:"max_idle_conns" toml:"max_idle_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime" json:"conn_max_lifetime" toml:"conn_max_lifetime"`
	ConnMaxIdleTime int `yaml:"conn_max_idle_time" json:"conn_max_idle_time" toml:"conn_max_idle_time"`
}

// Validate sets defaults and validates PoolConfig.
func (p *PoolConfig) Validate() error {
	if p.MaxOpenConns <= 0 {
		p.MaxOpenConns = 50
	}
	if p.MaxIdleConns <= 0 {
		p.MaxIdleConns = 10
	}
	if p.ConnMaxLifetime <= 0 {
		p.ConnMaxLifetime = 1800
	}
	if p.ConnMaxIdleTime <= 0 {
		p.ConnMaxIdleTime = 600
	}
	// Cap MaxIdleConns at MaxOpenConns
	if p.MaxIdleConns > p.MaxOpenConns {
		p.MaxIdleConns = p.MaxOpenConns
	}
	return nil
}

// ConnMaxLifetimeDuration returns ConnMaxLifetime as time.Duration.
func (p *PoolConfig) ConnMaxLifetimeDuration() time.Duration {
	return time.Duration(p.ConnMaxLifetime) * time.Second
}

// ConnMaxIdleTimeDuration returns ConnMaxIdleTime as time.Duration.
func (p *PoolConfig) ConnMaxIdleTimeDuration() time.Duration {
	return time.Duration(p.ConnMaxIdleTime) * time.Second
}
