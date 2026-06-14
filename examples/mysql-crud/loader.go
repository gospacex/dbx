package main

import (
	"fmt"
	"os"

	"github.com/gospacex/dbx/config"

	"gopkg.in/yaml.v3"
)

// loadSingleFromYAML loads the mysql.example.yaml-shaped file and
// returns the resolved DBConfig together with the mqx-style trace
// config. The mqx-style trace is translated to dbx internal schema
// downstream in openSingleFromYAML.
func loadSingleFromYAML(path string) (config.DBConfig, *mqxTraceConfig, error) {
	raw, err := readTopSections(path)
	if err != nil {
		return nil, nil, err
	}
	// Reuse the dbx loader for the driver section so the parsing /
	// validation rules match the rest of the framework.
	if raw.MySQL == nil {
		return nil, nil, fmt.Errorf("loader: missing mysql section in %s", path)
	}
	if err := raw.MySQL.Validate(); err != nil {
		return nil, nil, fmt.Errorf("loader: validate mysql: %w", err)
	}
	return raw.MySQL, raw.Trace, nil
}

// loadClusterFromYAML loads the cluster.example.yaml-shaped file and
// returns the resolved ClusterConfig together with the mqx-style
// trace config. The translation happens in openClusterFromYAML.
func loadClusterFromYAML(path string) (*config.ClusterConfig, *mqxTraceConfig, error) {
	raw, err := readTopSections(path)
	if err != nil {
		return nil, nil, err
	}
	if raw.Cluster == nil {
		return nil, nil, fmt.Errorf("loader: missing cluster section in %s", path)
	}
	if err := raw.Cluster.Validate(); err != nil {
		return nil, nil, fmt.Errorf("loader: validate cluster: %w", err)
	}
	return raw.Cluster, raw.Trace, nil
}

// rawFile is the on-disk yaml shape. The driver/cluster section is
// decoded by dbx, but the trace section uses the mqx-standard field
// names so this example does NOT re-use config.TracingConfig (whose
// yaml tags would force `service`, `http/protobuf`, `parentbased_…`).
type rawFile struct {
	MySQL   *config.MySQLConfig   `yaml:"mysql"`
	Cluster *config.ClusterConfig `yaml:"cluster"`
	Trace   *mqxTraceConfig       `yaml:"trace"`
	Logger  *loggerConfig         `yaml:"logger"`
}

func readTopSections(path string) (*rawFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loader: read %s: %w", path, err)
	}
	var out rawFile
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("loader: parse yaml %s: %w", path, err)
	}
	return &out, nil
}
