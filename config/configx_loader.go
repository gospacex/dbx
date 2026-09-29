package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gospacex/configx"
)

// configx_loader.go
// 从 configx 句柄读取 SQL 数据库配置，与 configx 默认配置文件
// （configx/configs/config.enterprise.yaml 的 xmiddleware.sqlx 段）对齐。
//
// 读取路径优先级:
//  1. xmiddleware.sqlx.databases.<name>   （configx 默认配置）
//  2. xmiddleware.dbx.databases.<name>    （别名兼容）
//  3. xmiddleware.sqlx.instances.<name>   （instances 风格兼容）
//
// 企业默认配置中 SQL 实例通常只给 dsn 与连接池字段，本 loader 会：
//   - dsn 存在且 host/username 缺失时，按 MySQL DSN 规范解析补全（mysql/mariadb/tidb 适用）
//   - conn_max_lifetime / conn_max_idle_time 同时兼容 "5m" 时长串与整数秒
//   - enable_trace 映射为 TracingConfig.Enabled（tracing.* 子段优先）

// sqlxInstancePrefix 解析实例配置路径（见文件头注释）
func sqlxInstancePrefix(cc configx.Config, name string) string {
	if name == "" {
		name = "default"
	}
	candidates := []string{
		"xmiddleware.sqlx.databases",
		"xmiddleware.dbx.databases",
		"xmiddleware.sqlx.instances",
		"xmiddleware.dbx.instances",
	}
	for _, section := range candidates {
		p := section + "." + name
		if cc.IsSet(p) {
			return p
		}
	}
	return "xmiddleware.sqlx.databases." + name
}

// durationSeconds 兼容时长串与整数秒两种写法
func durationSeconds(cc configx.Config, key string) int {
	if !cc.IsSet(key) {
		return 0
	}
	switch v := cc.Get(key).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return int(d.Seconds())
		}
	}
	return int(cc.GetDuration(key).Seconds())
}

// applyConfigxBase 把 sqlx 实例的公共字段填入 BaseDBConfig
func applyConfigxBase(cc configx.Config, prefix string, base *BaseDBConfig) {
	base.Host = cc.GetString(prefix + ".host")
	base.Port = cc.GetInt(prefix + ".port")
	base.Username = cc.GetString(prefix + ".username")
	base.Password = cc.GetString(prefix + ".password")
	base.Database = cc.GetString(prefix + ".database")

	// dsn 兜底：企业默认配置只给 dsn 时，按 MySQL DSN 规范解析
	dsn := cc.GetString(prefix + ".dsn")
	if dsn != "" && base.Host == "" && base.Username == "" {
		if parsed, err := mysql.ParseDSN(dsn); err == nil {
			base.Host = parsed.Addr
			if i := strings.LastIndex(base.Host, ":"); i >= 0 {
				host := base.Host[:i]
				port := base.Host[i+1:]
				base.Host = host
				fmt.Sscanf(port, "%d", &base.Port)
			}
			base.Username = parsed.User
			base.Password = parsed.Passwd
			base.Database = parsed.DBName
		}
	}

	base.Pool = &PoolConfig{
		MaxOpenConns:    cc.GetInt(prefix + ".max_open_conns"),
		MaxIdleConns:    cc.GetInt(prefix + ".max_idle_conns"),
		ConnMaxLifetime: durationSeconds(cc, prefix+".conn_max_lifetime"),
		ConnMaxIdleTime: durationSeconds(cc, prefix+".conn_max_idle_time"),
	}
}

// TracingFromConfigx 读取 tracing 子段（缺省时以 enable_trace 作为开关）。
// 两者都未配置时返回 nil。导出供 dbsql 的 configx 入口复用，保证
// configx 路径与文件路径的 tracing 语义完全一致。
func TracingFromConfigx(cc configx.Config, prefix string) *TracingConfig {
	tp := prefix + ".tracing"
	if !cc.IsSet(tp) && !cc.IsSet(prefix+".enable_trace") {
		return nil
	}
	tc := &TracingConfig{
		Enabled:      cc.GetBool(tp + ".enabled"),
		Service:      cc.GetString(tp + ".service"),
		Exporter:     cc.GetString(tp + ".exporter"),
		Endpoint:     cc.GetString(tp + ".endpoint"),
		SamplerType:  cc.GetString(tp + ".sampler_type"),
		SamplerRatio: cc.GetFloat64(tp + ".sampler_ratio"),
		Mode:         cc.GetString(tp + ".mode"),
		Topic:        cc.GetString(tp + ".topic"),
		Stream:       cc.GetString(tp + ".stream"),
	}
	// 开关解析：tracing 子段里显式写了 enabled 时以它为准；否则一律回落到
	// 扁平的 enable_trace。
	//
	// 中间的 else-if 是修一个静默失效：用户常常只把 exporter 细节放进
	// tracing 段而把开关留在顶层，形如
	//
	//	tracing:
	//	  exporter: jaeger
	//	  endpoint: localhost:4318
	//	enable_trace: true            # ← 期望生效
	//
	// 旧逻辑只看 tracing.enabled（缺省 false），enable_trace 被无声吞掉，
	// 表现是"配了却不埋点、也不报错"。现在只要 tracing.enabled 没显式写，
	// 就沿用 enable_trace。
	switch {
	case !cc.IsSet(tp):
		tc.Enabled = cc.GetBool(prefix + ".enable_trace")
	case !cc.IsSet(tp + ".enabled"):
		tc.Enabled = cc.GetBool(prefix + ".enable_trace")
	}
	return tc
}

// validateInstance 统一收尾：填充池/追踪默认值并校验
func validateInstance(base *BaseDBConfig, tc *TracingConfig) (*TracingConfig, error) {
	base.Pool.Validate()
	if tc != nil {
		if err := tc.Validate(); err != nil {
			return nil, err
		}
	}
	return tc, nil
}

// LoadDBConfigFromConfigx 按 driver 类型从 configx 读取数据库配置
func LoadDBConfigFromConfigx(cc configx.Config, driver, name string) (DBConfig, *TracingConfig, error) {
	switch strings.ToLower(driver) {
	case "mysql":
		return LoadMySQLFromConfigx(cc, name)
	case "postgres", "postgresql":
		return LoadPostgresFromConfigx(cc, name)
	case "tidb":
		return LoadTiDBFromConfigx(cc, name)
	case "mariadb":
		return LoadMariaDBFromConfigx(cc, name)
	case "gaussdb":
		return LoadGaussDBFromConfigx(cc, name)
	case "mssql":
		return LoadMSSQLFromConfigx(cc, name)
	case "oracle":
		return LoadOracleFromConfigx(cc, name)
	case "sqlite":
		return LoadSQLiteFromConfigx(cc, name)
	default:
		return nil, nil, fmt.Errorf("configx loader: unsupported driver %q", driver)
	}
}

// LoadMySQLFromConfigx 从 configx 读取 MySQL 配置
func LoadMySQLFromConfigx(cc configx.Config, name string) (*MySQLConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &MySQLConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.Charset = cc.GetString(p + ".charset")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadPostgresFromConfigx 从 configx 读取 Postgres 配置
func LoadPostgresFromConfigx(cc configx.Config, name string) (*PostgresConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &PostgresConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.SSLMode = cc.GetString(p + ".ssl_mode")
	c.TimeZone = cc.GetString(p + ".timezone")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadTiDBFromConfigx 从 configx 读取 TiDB 配置
func LoadTiDBFromConfigx(cc configx.Config, name string) (*TiDBConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &TiDBConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.Charset = cc.GetString(p + ".charset")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadMariaDBFromConfigx 从 configx 读取 MariaDB 配置
func LoadMariaDBFromConfigx(cc configx.Config, name string) (*MariaDBConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &MariaDBConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.Charset = cc.GetString(p + ".charset")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadGaussDBFromConfigx 从 configx 读取 GaussDB 配置
func LoadGaussDBFromConfigx(cc configx.Config, name string) (*GaussDBConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &GaussDBConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.SSLMode = cc.GetString(p + ".ssl_mode")
	c.Schema = cc.GetString(p + ".schema")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadMSSQLFromConfigx 从 configx 读取 MSSQL 配置
func LoadMSSQLFromConfigx(cc configx.Config, name string) (*MSSQLConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &MSSQLConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.Instance = cc.GetString(p + ".instance")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadOracleFromConfigx 从 configx 读取 Oracle 配置
func LoadOracleFromConfigx(cc configx.Config, name string) (*OracleConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &OracleConfig{}
	applyConfigxBase(cc, p, &c.BaseDBConfig)
	c.ServiceName = cc.GetString(p + ".service_name")
	c.SID = cc.GetString(p + ".sid")
	tc, err := validateInstance(&c.BaseDBConfig, TracingFromConfigx(cc, p))
	if err != nil {
		return nil, nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}

// LoadSQLiteFromConfigx 从 configx 读取 SQLite 配置
func LoadSQLiteFromConfigx(cc configx.Config, name string) (*SQLiteConfig, *TracingConfig, error) {
	p := sqlxInstancePrefix(cc, name)
	c := &SQLiteConfig{}
	c.Path = cc.GetString(p + ".path")
	if c.Path == "" {
		c.Path = cc.GetString(p + ".dsn")
	}
	c.Pool = &PoolConfig{
		MaxOpenConns:    cc.GetInt(p + ".max_open_conns"),
		MaxIdleConns:    cc.GetInt(p + ".max_idle_conns"),
		ConnMaxLifetime: durationSeconds(cc, p+".conn_max_lifetime"),
		ConnMaxIdleTime: durationSeconds(cc, p+".conn_max_idle_time"),
	}
	c.Pool.Validate()
	tc := TracingFromConfigx(cc, p)
	if tc != nil {
		if err := tc.Validate(); err != nil {
			return nil, nil, err
		}
	}
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	return c, tc, nil
}
