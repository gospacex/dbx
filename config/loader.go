package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type fileWrapper struct {
	MySQL    *MySQLConfig    `yaml:"mysql" json:"mysql" toml:"mysql"`
	Postgres *PostgresConfig `yaml:"postgres" json:"postgres" toml:"postgres"`
	TiDB     *TiDBConfig     `yaml:"tidb" json:"tidb" toml:"tidb"`
	MariaDB  *MariaDBConfig  `yaml:"mariadb" json:"mariadb" toml:"mariadb"`
	GaussDB  *GaussDBConfig  `yaml:"gaussdb" json:"gaussdb" toml:"gaussdb"`
	MSSQL    *MSSQLConfig    `yaml:"mssql" json:"mssql" toml:"mssql"`
	Oracle   *OracleConfig   `yaml:"oracle" json:"oracle" toml:"oracle"`
	SQLite   *SQLiteConfig   `yaml:"sqlite" json:"sqlite" toml:"sqlite"`
	Cluster  *ClusterConfig  `yaml:"cluster" json:"cluster" toml:"cluster"`
	Tracing  *TracingConfig  `yaml:"tracing" json:"tracing" toml:"tracing"`
}

// validator is the only contract loadSection needs from a config section.
type validator interface {
	Validate() error
}

// loadSection decodes path into fileWrapper, extracts one section with pick,
// and validates the section plus the optional tracing section. pick must
// return a typed nil pointer when the section is absent.
func loadSection[T validator](path, section string, pick func(*fileWrapper) T) (T, *TracingConfig, error) {
	var zero T
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return zero, nil, err
	}
	cfg := pick(w)
	if isNilPtr(cfg) {
		return zero, nil, fmt.Errorf("loader: missing %s section in %s", section, path)
	}
	if err := cfg.Validate(); err != nil {
		return zero, nil, err
	}
	if err := validateTracing(w.Tracing); err != nil {
		return zero, nil, err
	}
	return cfg, w.Tracing, nil
}

func validateTracing(tc *TracingConfig) error {
	if tc != nil {
		return tc.Validate()
	}
	return nil
}

// driverSection registers one driver's filename aliases, its LoadXxx
// entrypoint, and a section picker for Load's content-based fallback.
// Adding a driver means adding one entry here plus one public LoadXxx func.
type driverSection struct {
	name    string
	aliases []string
	load    func(path string) (DBConfig, *TracingConfig, error)
	pick    func(w *fileWrapper) DBConfig
}

// dbConfigOrNil converts a typed nil *XxxConfig to a nil DBConfig so
// pickers can be tested with a plain == nil.
func dbConfigOrNil[T DBConfig](c T) DBConfig {
	if isNilPtr(c) {
		return nil
	}
	return c
}

// isNilPtr reports whether v is a typed nil pointer. A typed nil stored in
// an interface is not == nil, hence the reflection. All config types in
// this package are pointer-based, so IsNil never panics here.
func isNilPtr[T any](v T) bool {
	return reflect.ValueOf(v).IsNil()
}

var driverSections = []driverSection{
	{"mysql", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadMySQL(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.MySQL) }},
	{"postgres", []string{"postgresql"},
		func(p string) (DBConfig, *TracingConfig, error) { return LoadPostgres(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.Postgres) }},
	{"tidb", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadTiDB(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.TiDB) }},
	{"mariadb", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadMariaDB(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.MariaDB) }},
	{"gaussdb", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadGaussDB(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.GaussDB) }},
	{"mssql", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadMSSQL(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.MSSQL) }},
	{"oracle", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadOracle(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.Oracle) }},
	{"sqlite", nil,
		func(p string) (DBConfig, *TracingConfig, error) { return LoadSQLite(p) },
		func(w *fileWrapper) DBConfig { return dbConfigOrNil(w.SQLite) }},
}

// LoadMySQL loads a MySQL config from a YAML/JSON/TOML file.
func LoadMySQL(path string) (*MySQLConfig, *TracingConfig, error) {
	return loadSection(path, "mysql", func(w *fileWrapper) *MySQLConfig { return w.MySQL })
}

// LoadPostgres loads a Postgres config from a file.
func LoadPostgres(path string) (*PostgresConfig, *TracingConfig, error) {
	return loadSection(path, "postgres", func(w *fileWrapper) *PostgresConfig { return w.Postgres })
}

// LoadTiDB loads a TiDB config from a file.
func LoadTiDB(path string) (*TiDBConfig, *TracingConfig, error) {
	return loadSection(path, "tidb", func(w *fileWrapper) *TiDBConfig { return w.TiDB })
}

// LoadMariaDB loads a MariaDB config from a file.
func LoadMariaDB(path string) (*MariaDBConfig, *TracingConfig, error) {
	return loadSection(path, "mariadb", func(w *fileWrapper) *MariaDBConfig { return w.MariaDB })
}

// LoadGaussDB loads a GaussDB config from a file.
func LoadGaussDB(path string) (*GaussDBConfig, *TracingConfig, error) {
	return loadSection(path, "gaussdb", func(w *fileWrapper) *GaussDBConfig { return w.GaussDB })
}

// LoadMSSQL loads a MSSQL config from a file.
func LoadMSSQL(path string) (*MSSQLConfig, *TracingConfig, error) {
	return loadSection(path, "mssql", func(w *fileWrapper) *MSSQLConfig { return w.MSSQL })
}

// LoadOracle loads an Oracle config from a file.
func LoadOracle(path string) (*OracleConfig, *TracingConfig, error) {
	return loadSection(path, "oracle", func(w *fileWrapper) *OracleConfig { return w.Oracle })
}

// LoadSQLite loads a SQLite config from a file.
func LoadSQLite(path string) (*SQLiteConfig, *TracingConfig, error) {
	return loadSection(path, "sqlite", func(w *fileWrapper) *SQLiteConfig { return w.SQLite })
}

// LoadCluster loads a ClusterConfig from a file.
func LoadCluster(path string) (*ClusterConfig, *TracingConfig, error) {
	return loadSection(path, "cluster", func(w *fileWrapper) *ClusterConfig { return w.Cluster })
}

// Load auto-detects format based on file basename and returns a DBConfig.
// If the basename doesn't match a known driver, falls back to dispatching by
// the first populated section in the file (so db.yaml with `mysql:` block works
// regardless of filename).
func Load(path string) (DBConfig, *TracingConfig, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, d := range driverSections {
		if d.name == name || contains(d.aliases, name) {
			return d.load(path)
		}
	}
	// Filename didn't match; dispatch by populated section.
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if err := validateTracing(w.Tracing); err != nil {
		return nil, nil, err
	}
	for _, d := range driverSections {
		if cfg := d.pick(w); cfg != nil {
			if err := cfg.Validate(); err != nil {
				return nil, nil, err
			}
			return cfg, w.Tracing, nil
		}
	}
	return nil, nil, fmt.Errorf("loader: no driver section found in %s", path)
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func loadFile(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("loader: read %s: %w", path, err)
	}
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, out); err != nil {
			return fmt.Errorf("loader: parse yaml %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("loader: parse json %s: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, out); err != nil {
			return fmt.Errorf("loader: parse toml %s: %w", path, err)
		}
	default:
		return fmt.Errorf("loader: unsupported file extension %q", ext)
	}
	return nil
}
