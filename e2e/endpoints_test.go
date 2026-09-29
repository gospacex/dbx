//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gospacex/configx"
)

// readYAMLConfig writes body to a temp config.yaml and returns a configx
// handle for it. Callers own the handle; it is closed on test cleanup.
func readYAMLConfigFile(t *testing.T, body string) configx.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := configx.ReadConfig(configx.ConfigOptions{ConfigFile: path})
	if err != nil {
		t.Fatalf("configx.ReadConfig error: %v", err)
	}
	t.Cleanup(func() { _ = cfg.Close() })
	return cfg
}

// Every service endpoint used by this suite is resolved through the helpers
// below. The defaults match the tutorial docker-compose.yml shipped with the
// project, so an unmodified checkout keeps working; the environment overrides
// exist because that compose file (and most local setups) remap host ports to
// avoid collisions with a system install — on this machine MySQL is on 13306,
// Postgres on 15432 and Redis on 46379, so the hard-coded defaults fail with
// "connection refused" for reasons that have nothing to do with dbx.
//
// Recognised variables:
//
//	DBX_E2E_MYSQL_HOST    DBX_E2E_MYSQL_PORT    DBX_E2E_MYSQL_USER
//	DBX_E2E_MYSQL_PASS    DBX_E2E_MYSQL_DB
//	DBX_E2E_PG_HOST       DBX_E2E_PG_PORT       DBX_E2E_PG_USER
//	DBX_E2E_PG_PASS       DBX_E2E_PG_DB
//	DBX_E2E_REDIS_ADDR    DBX_E2E_REDIS_PASS
//	DBX_E2E_KAFKA_BROKER
//	DBX_E2E_JAEGER_OTLP   DBX_E2E_JAEGER_QUERY
const (
	defaultMySQLHost = "127.0.0.1"
	defaultMySQLPort = 3306
	defaultMySQLUser = "root"
	defaultMySQLPass = "rootpass"
	defaultMySQLDB   = "testdb"

	defaultPGHost = "127.0.0.1"
	defaultPGPort = 5432
	defaultPGUser = "postgres"
	defaultPGPass = "postgres"
	defaultPGDB   = "postgres"

	defaultRedisAddr = "127.0.0.1:6379"
	defaultRedisPass = "redis123456"

	defaultKafkaBroker = "127.0.0.1:9092"

	defaultJaegerOTLP  = "127.0.0.1:4318"
	defaultJaegerQuery = "http://127.0.0.1:16686"
)

// envOr returns the value of key, or def when the variable is unset or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envPort returns the port in key parsed as an integer, or def when the
// variable is unset, empty or not a valid port. A malformed override falls
// back to the default instead of failing the whole suite, so a typo shows up
// as a "connection refused" against the documented default rather than a
// strconv error with no context.
func envPort(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p < 65536 {
			return p
		}
	}
	return def
}

type sqlEndpoint struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func mysqlEndpoint() sqlEndpoint {
	return sqlEndpoint{
		Host:     envOr("DBX_E2E_MYSQL_HOST", defaultMySQLHost),
		Port:     envPort("DBX_E2E_MYSQL_PORT", defaultMySQLPort),
		Username: envOr("DBX_E2E_MYSQL_USER", defaultMySQLUser),
		Password: envOr("DBX_E2E_MYSQL_PASS", defaultMySQLPass),
		Database: envOr("DBX_E2E_MYSQL_DB", defaultMySQLDB),
	}
}

func pgEndpoint() sqlEndpoint {
	return sqlEndpoint{
		Host:     envOr("DBX_E2E_PG_HOST", defaultPGHost),
		Port:     envPort("DBX_E2E_PG_PORT", defaultPGPort),
		Username: envOr("DBX_E2E_PG_USER", defaultPGUser),
		Password: envOr("DBX_E2E_PG_PASS", defaultPGPass),
		Database: envOr("DBX_E2E_PG_DB", defaultPGDB),
	}
}

// mysqlDSN / pgDSN are derived from the same overrides as the structured
// configs, so a run cannot end up pointing the DSN form at one server and the
// struct form at another.
func mysqlDSN() string {
	e := mysqlEndpoint()
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		e.Username, e.Password, e.Host, e.Port, e.Database)
}

func pgDSN() string {
	e := pgEndpoint()
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		e.Host, e.Port, e.Username, e.Password, e.Database)
}

func redisAddr() string   { return envOr("DBX_E2E_REDIS_ADDR", defaultRedisAddr) }
func redisPass() string   { return envOr("DBX_E2E_REDIS_PASS", defaultRedisPass) }
func kafkaBroker() string { return envOr("DBX_E2E_KAFKA_BROKER", defaultKafkaBroker) }

// jaegerOTLPEndpoint is host:port for the OTLP/HTTP receiver;
// jaegerQueryBase is the HTTP base URL of the Jaeger query API.
func jaegerOTLPEndpoint() string { return envOr("DBX_E2E_JAEGER_OTLP", defaultJaegerOTLP) }
func jaegerQueryBase() string    { return envOr("DBX_E2E_JAEGER_QUERY", defaultJaegerQuery) }
