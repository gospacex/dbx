package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// LoadMySQL loads a MySQL config from a YAML/JSON/TOML file.
func LoadMySQL(path string) (*MySQLConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.MySQL == nil {
		return nil, nil, fmt.Errorf("loader: missing mysql section in %s", path)
	}
	if err := w.MySQL.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.MySQL, w.Tracing, nil
}

// LoadPostgres loads a Postgres config from a file.
func LoadPostgres(path string) (*PostgresConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.Postgres == nil {
		return nil, nil, fmt.Errorf("loader: missing postgres section in %s", path)
	}
	if err := w.Postgres.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.Postgres, w.Tracing, nil
}

// LoadTiDB loads a TiDB config from a file.
func LoadTiDB(path string) (*TiDBConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.TiDB == nil {
		return nil, nil, fmt.Errorf("loader: missing tidb section in %s", path)
	}
	if err := w.TiDB.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.TiDB, w.Tracing, nil
}

// LoadMariaDB loads a MariaDB config from a file.
func LoadMariaDB(path string) (*MariaDBConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.MariaDB == nil {
		return nil, nil, fmt.Errorf("loader: missing mariadb section in %s", path)
	}
	if err := w.MariaDB.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.MariaDB, w.Tracing, nil
}

// LoadGaussDB loads a GaussDB config from a file.
func LoadGaussDB(path string) (*GaussDBConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.GaussDB == nil {
		return nil, nil, fmt.Errorf("loader: missing gaussdb section in %s", path)
	}
	if err := w.GaussDB.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.GaussDB, w.Tracing, nil
}

// LoadMSSQL loads a MSSQL config from a file.
func LoadMSSQL(path string) (*MSSQLConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.MSSQL == nil {
		return nil, nil, fmt.Errorf("loader: missing mssql section in %s", path)
	}
	if err := w.MSSQL.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.MSSQL, w.Tracing, nil
}

// LoadOracle loads an Oracle config from a file.
func LoadOracle(path string) (*OracleConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.Oracle == nil {
		return nil, nil, fmt.Errorf("loader: missing oracle section in %s", path)
	}
	if err := w.Oracle.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.Oracle, w.Tracing, nil
}

// LoadSQLite loads a SQLite config from a file.
func LoadSQLite(path string) (*SQLiteConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.SQLite == nil {
		return nil, nil, fmt.Errorf("loader: missing sqlite section in %s", path)
	}
	if err := w.SQLite.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.SQLite, w.Tracing, nil
}

// LoadCluster loads a ClusterConfig from a file.
func LoadCluster(path string) (*ClusterConfig, *TracingConfig, error) {
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.Cluster == nil {
		return nil, nil, fmt.Errorf("loader: missing cluster section in %s", path)
	}
	if err := w.Cluster.Validate(); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	return w.Cluster, w.Tracing, nil
}

// Load auto-detects format based on file basename and returns a DBConfig.
// If the basename doesn't match a known driver, falls back to dispatching by
// the first populated section in the file (so db.yaml with `mysql:` block works
// regardless of filename).
func Load(path string) (DBConfig, *TracingConfig, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch name {
	case "mysql":
		return LoadMySQL(path)
	case "postgres", "postgresql":
		return LoadPostgres(path)
	case "tidb":
		return LoadTiDB(path)
	case "mariadb":
		return LoadMariaDB(path)
	case "gaussdb":
		return LoadGaussDB(path)
	case "mssql":
		return LoadMSSQL(path)
	case "oracle":
		return LoadOracle(path)
	case "sqlite":
		return LoadSQLite(path)
	}
	// Filename didn't match; dispatch by populated section.
	w := &fileWrapper{}
	if err := loadFile(path, w); err != nil {
		return nil, nil, err
	}
	if w.Tracing != nil {
		if err := w.Tracing.Validate(); err != nil {
			return nil, nil, err
		}
	}
	switch {
	case w.MySQL != nil:
		if err := w.MySQL.Validate(); err != nil {
			return nil, nil, err
		}
		return w.MySQL, w.Tracing, nil
	case w.Postgres != nil:
		if err := w.Postgres.Validate(); err != nil {
			return nil, nil, err
		}
		return w.Postgres, w.Tracing, nil
	case w.TiDB != nil:
		if err := w.TiDB.Validate(); err != nil {
			return nil, nil, err
		}
		return w.TiDB, w.Tracing, nil
	case w.MariaDB != nil:
		if err := w.MariaDB.Validate(); err != nil {
			return nil, nil, err
		}
		return w.MariaDB, w.Tracing, nil
	case w.GaussDB != nil:
		if err := w.GaussDB.Validate(); err != nil {
			return nil, nil, err
		}
		return w.GaussDB, w.Tracing, nil
	case w.MSSQL != nil:
		if err := w.MSSQL.Validate(); err != nil {
			return nil, nil, err
		}
		return w.MSSQL, w.Tracing, nil
	case w.Oracle != nil:
		if err := w.Oracle.Validate(); err != nil {
			return nil, nil, err
		}
		return w.Oracle, w.Tracing, nil
	case w.SQLite != nil:
		if err := w.SQLite.Validate(); err != nil {
			return nil, nil, err
		}
		return w.SQLite, w.Tracing, nil
	}
	return nil, nil, fmt.Errorf("loader: no driver section found in %s", path)
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
