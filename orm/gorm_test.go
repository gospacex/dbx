package orm_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/orm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeDBConfig implements config.DBConfig for testing orm.Open
type fakeDBConfig struct {
	driver string
	dsn    string
}

func (f *fakeDBConfig) DriverName() string { return f.driver }
func (f *fakeDBConfig) DSN() string        { return f.dsn }
func (f *fakeDBConfig) Validate() error    { return nil }
func (f *fakeDBConfig) GetPool() *config.PoolConfig      { return nil }
func (f *fakeDBConfig) GetTracing() *config.TracingConfig { return nil }

func TestDialector_Mysql(t *testing.T) {
	d, err := orm.Dialector("mysql", "user:pass@tcp(localhost:3306)/db")
	if err != nil {
		t.Fatalf("Dialector(mysql) error: %v", err)
	}
	if d == nil {
		t.Fatal("dialector should not be nil")
	}
	if _, ok := d.(*mysql.Dialector); !ok {
		t.Fatalf("expected *mysql.Dialector, got %T", d)
	}
}

func TestDialector_Postgres(t *testing.T) {
	d, err := orm.Dialector("postgres", "host=localhost user=postgres dbname=test")
	if err != nil {
		t.Fatalf("Dialector(postgres) error: %v", err)
	}
	if _, ok := d.(*postgres.Dialector); !ok {
		t.Fatalf("expected *postgres.Dialector, got %T", d)
	}
}

func TestDialector_Mssql(t *testing.T) {
	d, err := orm.Dialector("mssql", "sqlserver://user:pass@localhost:1433/db")
	if err != nil {
		t.Fatalf("Dialector(mssql) error: %v", err)
	}
	if _, ok := d.(*sqlserver.Dialector); !ok {
		t.Fatalf("expected *sqlserver.Dialector, got %T", d)
	}
}

func TestDialector_Sqlite(t *testing.T) {
	d, err := orm.Dialector("sqlite", "test.db")
	if err != nil {
		t.Fatalf("Dialector(sqlite) error: %v", err)
	}
	if _, ok := d.(*sqlite.Dialector); !ok {
		t.Fatalf("expected *sqlite.Dialector, got %T", d)
	}
}

func TestDialector_Unsupported(t *testing.T) {
	_, err := orm.Dialector("invalid", "")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestSupportedDrivers(t *testing.T) {
	drivers := orm.SupportedDrivers()
	if len(drivers) == 0 {
		t.Fatal("SupportedDrivers should not be empty")
	}
}

func TestOpen_Sqlite(t *testing.T) {
	cfg, err := orm.Open(&fakeDBConfig{
		driver: "sqlite",
		dsn:    ":memory:",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Open(sqlite) error: %v", err)
	}
	if cfg == nil {
		t.Fatal("db should not be nil")
	}
	sqlDB, err := cfg.DB()
	if err != nil {
		t.Fatalf("cfg.DB() error: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping() error: %v", err)
	}
	sqlDB.Close()
}
