package config

import "fmt"

var supportedClusterDrivers = map[string]bool{
	"mysql":    true,
	"tidb":     true,
	"mariadb":  true,
	"postgres": true,
	"gaussdb":  true,
	"mssql":    true,
	"oracle":   true,
	"sqlite":   true,
}

const (
	LoadBalanceRoundRobin = "round_robin"
	LoadBalanceRandom     = "random"
)

// ClusterConfig holds configuration for cluster (read/write splitting) access.
type ClusterConfig struct {
	Driver      string         `yaml:"driver" json:"driver" toml:"driver"`
	Sources     []NodeConfig   `yaml:"sources" json:"sources" toml:"sources"`
	Replicas    []NodeConfig   `yaml:"replicas,omitempty" json:"replicas,omitempty" toml:"replicas,omitempty"`
	LoadBalance string         `yaml:"load_balance" json:"load_balance" toml:"load_balance"`
	Pool        *PoolConfig    `yaml:"pool,omitempty" json:"pool,omitempty" toml:"pool,omitempty"`
	Tracing     *TracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty" toml:"tracing,omitempty"`
}

// NodeConfig holds a single database node connection info for cluster use.
// Network fields are used by 7/8 drivers; Path is used by sqlite.
type NodeConfig struct {
	Host     string `yaml:"host" json:"host" toml:"host"`
	Port     int    `yaml:"port" json:"port" toml:"port"`
	Username string `yaml:"username" json:"username" toml:"username"`
	Password string `yaml:"password" json:"-" toml:"password"`
	Database string `yaml:"database" json:"database" toml:"database"`
	// Path is the file path for sqlite cluster nodes (reuses Driver=sqlite).
	Path string `yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`
	// Expanded fields per driver
	Instance    string `yaml:"instance,omitempty" json:"instance,omitempty" toml:"instance,omitempty"`
	ServiceName string `yaml:"service_name,omitempty" json:"service_name,omitempty" toml:"service_name,omitempty"`
	SID         string `yaml:"sid,omitempty" json:"sid,omitempty" toml:"sid,omitempty"`
	SSLMode     string `yaml:"ssl_mode,omitempty" json:"ssl_mode,omitempty" toml:"ssl_mode,omitempty"`
	Schema      string `yaml:"schema,omitempty" json:"schema,omitempty" toml:"schema,omitempty"`
}

// Validate validates ClusterConfig, setting defaults.
func (cc *ClusterConfig) Validate() error {
	if !supportedClusterDrivers[cc.Driver] {
		return fmt.Errorf("cluster: unsupported driver %q", cc.Driver)
	}
	if len(cc.Sources) == 0 {
		return fmt.Errorf("cluster: at least one source is required")
	}
	if cc.LoadBalance == "" {
		cc.LoadBalance = LoadBalanceRoundRobin
	}
	if cc.LoadBalance != LoadBalanceRoundRobin && cc.LoadBalance != LoadBalanceRandom {
		return fmt.Errorf("cluster: unsupported load_balance %q", cc.LoadBalance)
	}
	if cc.Pool == nil {
		cc.Pool = &PoolConfig{}
	}
	return cc.Pool.Validate()
}

// DSN builds a DSN for the given driver. The driver is the cluster-level driver
// (e.g. "mysql", "postgres", "mssql", "oracle", "sqlite"). Network drivers use
// the network fields; sqlite uses Path.
func (n *NodeConfig) DSN(driver string) string {
	switch driver {
	case "mysql", "tidb", "mariadb":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			n.Username, n.Password, n.Host, n.Port, n.Database)
	case "postgres", "gaussdb":
		s := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			n.Host, n.Port, n.Username, n.Password, n.Database, n.SSLMode)
		if n.Schema != "" {
			s += " search_path=" + n.Schema
		}
		return s
	case "mssql":
		if n.Instance != "" {
			return fmt.Sprintf("sqlserver://%s:%s@%s/%s?instance=%s",
				n.Username, n.Password, n.Host, n.Database, n.Instance)
		}
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d/%s",
			n.Username, n.Password, n.Host, n.Port, n.Database)
	case "oracle":
		if n.ServiceName != "" {
			return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
				n.Username, n.Password, n.Host, n.Port, n.ServiceName)
		}
		return fmt.Sprintf("oracle://%s:%s@%s:%d/%s?sid=%s",
			n.Username, n.Password, n.Host, n.Port, n.Database, n.SID)
	case "sqlite":
		return n.Path
	default:
		// Fall back to MySQL-style DSN for unknown drivers.
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", n.Username, n.Password, n.Host, n.Port, n.Database)
	}
}
