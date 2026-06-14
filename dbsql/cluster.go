package dbsql

import (
	"context"
	"fmt"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/orm"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// sourceConfig wraps a driver name and NodeConfig to satisfy orm.DialectorSource.
// It only implements the two methods orm.Open actually calls; pool and tracing
// are not applicable to a single cluster node and are configured at the
// ClusterConfig level (see cc.Pool / cc.Tracing in OpenCluster).
type sourceConfig struct {
	driver string
	node   config.NodeConfig
}

func (s *sourceConfig) DriverName() string { return s.driver }
func (s *sourceConfig) DSN() string        { return s.node.DSN(s.driver) }

// policyFromString maps a load balance string to a dbresolver policy.
// Supported: "round_robin" (default), "random".
func policyFromString(s string) dbresolver.Policy {
	switch s {
	case config.LoadBalanceRandom:
		return dbresolver.RandomPolicy{}
	case config.LoadBalanceRoundRobin, "":
		return dbresolver.RoundRobinPolicy()
	default:
		return dbresolver.RoundRobinPolicy()
	}
}

// OpenCluster opens a *gorm.DB with read/write splitting via dbresolver.
func OpenCluster(cc *config.ClusterConfig) (*gorm.DB, error) {
	if err := cc.Validate(); err != nil {
		return nil, fmt.Errorf("dbsql: cluster: %w", err)
	}

	// Open the primary connection from the first source.
	src := &sourceConfig{driver: cc.Driver, node: cc.Sources[0]}
	db, err := orm.Open(src)
	if err != nil {
		return nil, fmt.Errorf("dbsql: cluster: open: %w", err)
	}

	// Create dialectors for all sources and replicas.
	srcDialectors := make([]gorm.Dialector, len(cc.Sources))
	for i := range cc.Sources {
		d, err := orm.Dialector(cc.Driver, cc.Sources[i].DSN(cc.Driver))
		if err != nil {
			return nil, fmt.Errorf("dbsql: cluster: source %d: %w", i, err)
		}
		srcDialectors[i] = d
	}
	replicaDialectors := make([]gorm.Dialector, len(cc.Replicas))
	for i := range cc.Replicas {
		d, err := orm.Dialector(cc.Driver, cc.Replicas[i].DSN(cc.Driver))
		if err != nil {
			return nil, fmt.Errorf("dbsql: cluster: replica %d: %w", i, err)
		}
		replicaDialectors[i] = d
	}

	// Register dbresolver for read/write splitting.
	if err := db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  srcDialectors,
		Replicas: replicaDialectors,
		Policy:   policyFromString(cc.LoadBalance),
	})); err != nil {
		return nil, fmt.Errorf("dbsql: cluster: dbresolver: %w", err)
	}

	// Apply pool settings.
	if cc.Pool != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("dbsql: cluster: pool: %w", err)
		}
		sqlDB.SetMaxOpenConns(cc.Pool.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cc.Pool.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(cc.Pool.ConnMaxLifetimeDuration())
		sqlDB.SetConnMaxIdleTime(cc.Pool.ConnMaxIdleTimeDuration())
	}

	// Apply tracing.
	if err := ExtractTracingAndApply(context.Background(), cc.Tracing); err != nil {
		return nil, fmt.Errorf("dbsql: cluster: tracing: %w", err)
	}

	return db, nil
}

// OpenClusterPath loads a ClusterConfig from a file and opens a *gorm.DB.
func OpenClusterPath(path string) (*gorm.DB, error) {
	cc, tc, err := config.LoadCluster(path)
	if err != nil {
		return nil, fmt.Errorf("dbsql: cluster: load: %w", err)
	}
	if tc != nil {
		cc.Tracing = tc
	}
	return OpenCluster(cc)
}
