package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gorm.io/gorm"
)

// Exit codes (per tasks.md §7 acceptance):
//
//	0  success (including gorm.ErrRecordNotFound treated as soft fail)
//	2  configuration / argument error
//	3  connection error (MySQL unreachable, trace broker unreachable)
//	4  business error (duplicate primary key, constraint violation)
const (
	exitOK            = 0
	exitConfigError   = 2
	exitConnectError  = 3
	exitBusinessError = 4
)

// defaultConfig returns the conventional yaml path for a subcommand.
// `o` and `oc` use hardcoded configs and ignore the path; `p` and
// `pc` use yaml and require the user-supplied file to exist.
func defaultConfig(sub string) string {
	switch sub {
	case "p":
		return "mysql.yaml"
	case "pc":
		return "cluster.yaml"
	}
	return ""
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <subcommand> [-config <path>]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  o   open from hardcoded MySQL config         (dbsql.Open)\n")
	fmt.Fprintf(os.Stderr, "  p   open from yaml file                      (dbsql.OpenPath)\n")
	fmt.Fprintf(os.Stderr, "  oc  open cluster from hardcoded config       (dbsql.OpenCluster)\n")
	fmt.Fprintf(os.Stderr, "  pc  open cluster from yaml file              (dbsql.OpenClusterPath)\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitConfigError)
	}

	// Accept both `prog o` (positional) and `prog -sub o` (flag).
	first := os.Args[1]
	var sub string
	var configPath string
	if first == "-sub" {
		fs := flag.NewFlagSet("mysql-crud", flag.ContinueOnError)
		fs.StringVar(&sub, "sub", "o", "subcommand: o | p | oc | pc")
		fs.StringVar(&configPath, "config", "", "yaml config path (for p/pc)")
		if err := fs.Parse(os.Args[2:]); err != nil {
			usage()
			os.Exit(exitConfigError)
		}
	} else {
		sub = first
		fs := flag.NewFlagSet("mysql-crud", flag.ContinueOnError)
		fs.StringVar(&configPath, "config", defaultConfig(sub), "yaml config path (for p/pc)")
		// Ignore errors here so positional-only invocations work.
		_ = fs.Parse(os.Args[2:])
	}

	if configPath == "" {
		configPath = defaultConfig(sub)
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, shutdown, err := openForSubcommand(ctx, sub, configPath)
	if err != nil {
		// Distinguish config errors (REPLACE_ME, unknown subcommand,
		// missing -config) from connection errors. Both reach this
		// site; the user-facing exit code must reflect the cause.
		code := exitConnectError
		if isConfigError(err) {
			code = exitConfigError
		}
		fatal(code, "open", err)
	}
	defer func() {
		if shutdown != nil {
			if sErr := shutdown(ctx); sErr != nil {
				log.Printf("[FATAL] shutdown: %v", sErr)
			}
		}
	}()

	if err := runCRUDSequence(ctx, db); err != nil {
		code := exitBusinessError
		if isConnectionError(err) {
			code = exitConnectError
		}
		fatal(code, "crud", err)
	}

	log.Printf("[INFO] done")
	// Use `return` (not `os.Exit`) so the deferred shutdown closure
	// above runs. `os.Exit` skips deferred functions, which means the
	// BatchSpanProcessor's pending spans (5 CRUD + AutoMigrate) would
	// sit in its buffer and never reach jaeger — the BatchSpanProcessor
	// default interval is 5s and the example finishes in well under
	// that, so the process would die before the first batch tick.
	return
}

// openForSubcommand dispatches to one of the four dbx entries based
// on the subcommand. Returns (db, shutdown, err) on any failure so
// the caller can defer shutdown on the success path and log the
// failure cleanly on the error path.
func openForSubcommand(ctx context.Context, sub, configPath string) (*gorm.DB, func(context.Context) error, error) {
	switch sub {
	case "o":
		if hasReplaceMe(hardcodedMySQLConfig.Password) {
			return nil, nil, fmt.Errorf("config: password not set, replace REPLACE_ME in hardcoded config")
		}
		return openSingleFromStruct(ctx)
	case "p":
		if err := requireFile(configPath); err != nil {
			return nil, nil, err
		}
		if hasReplaceMeFile(configPath, "mysql") {
			return nil, nil, fmt.Errorf("config: password not set, replace REPLACE_ME in %s", configPath)
		}
		return openSingleFromYAML(ctx, configPath)
	case "oc":
		if hasReplaceMe(hardcodedClusterConfig.Sources[0].Password) {
			return nil, nil, fmt.Errorf("config: password not set, replace REPLACE_ME in hardcoded cluster config")
		}
		return openClusterFromStruct(ctx)
	case "pc":
		if err := requireFile(configPath); err != nil {
			return nil, nil, err
		}
		if hasReplaceMeFile(configPath, "cluster") {
			return nil, nil, fmt.Errorf("config: password not set, replace REPLACE_ME in %s", configPath)
		}
		return openClusterFromYAML(ctx, configPath)
	default:
		usage()
		return nil, nil, fmt.Errorf("config: unknown subcommand %q", sub)
	}
}

// runCRUDSequence performs the 5-step CRUD flow that the example
// demonstrates. Errors are wrapped with `fmt.Errorf("...: %w", err)`
// so the caller can distinguish classes via isConnectionError /
// isBusinessError.
//
// The whole flow is wrapped in a `user.crud` parent span so jaeger
// shows a single trace tree with the 5 (plus AutoMigrate) db.* spans
// as children, rather than 6 unrelated root spans. The GORM
// instrumentation picks the parent up automatically because it
// reads `Statement.Context` — every repo method already passes
// `ctx` through `db.WithContext(ctx)`.
func runCRUDSequence(ctx context.Context, db *gorm.DB) (err error) {
	tracer := otel.Tracer("github.com/gospacex/dbx/examples/mysql-crud")
	ctx, span := tracer.Start(ctx, "user.crud",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	// Named return `err` lets the deferred closure attach the
	// final error to the parent span without scattering the
	// RecordError/SetStatus calls through every return site.
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	if err = db.WithContext(ctx).AutoMigrate(&User{}); err != nil {
		err = fmt.Errorf("migrate: %w", err)
		return err
	}
	log.Printf("[INFO] migrated users table")

	repo := NewUserRepo(db)

	alice := &User{Name: "alice", Age: 30}
	if err = repo.CreateUser(ctx, alice); err != nil {
		err = fmt.Errorf("create: %w", err)
		return err
	}
	log.Printf("[INFO] created user id=%d", alice.ID)

	var got *User
	got, err = repo.GetUser(ctx, alice.ID)
	if err != nil {
		err = fmt.Errorf("get: %w", err)
		return err
	}
	log.Printf("[INFO] got user id=%d name=%s age=%d", got.ID, got.Name, got.Age)

	var list []User
	list, err = repo.ListUsers(ctx, 10)
	if err != nil {
		err = fmt.Errorf("list: %w", err)
		return err
	}
	log.Printf("[INFO] listed %d users", len(list))

	updated := *got
	updated.Age = 31
	if err = repo.UpdateUser(ctx, got.ID, &updated); err != nil {
		err = fmt.Errorf("update: %w", err)
		return err
	}
	log.Printf("[INFO] updated user id=%d age=%d", got.ID, updated.Age)

	if err = repo.DeleteUser(ctx, got.ID+1); err != nil {
		err = fmt.Errorf("delete: %w", err)
		return err
	}
	log.Printf("[INFO] deleted user id=%d (may be no-op)", got.ID+1)

	return nil
}

func fatal(code int, ctx string, err error) {
	log.Printf("[FATAL] %s: %v", ctx, err)
	os.Exit(code)
}

func requireFile(path string) error {
	if path == "" {
		return fmt.Errorf("config: missing -config <path>")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

func hasReplaceMe(pw string) bool {
	return strings.EqualFold(strings.TrimSpace(pw), "REPLACE_ME")
}

// hasReplaceMeFile scans the file for the marker. We do a textual
// match rather than parse-then-validate to keep the example small.
func hasReplaceMeFile(path, section string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, section+":") {
			inSection = true
			continue
		}
		if inSection && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			inSection = false
		}
		if inSection && strings.Contains(line, "REPLACE_ME") {
			return true
		}
	}
	return false
}

// isConnectionError reports whether err is the typical MySQL
// "can't connect" / "i/o timeout" class. Other errors are treated as
// business errors for exit-code purposes.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Driver / network classes.
	if errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}
	for _, needle := range []string{
		"connection refused",
		"i/o timeout",
		"no such host",
		"connect: ",
		"connectex:",
		"dial tcp",
		"bad connection",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// isConfigError reports whether err came from the configuration layer
// (REPLACE_ME marker, unknown subcommand, missing -config flag, file
// not found). The exit code for these is exitConfigError=2.
func isConfigError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"config:",
		"replace_replace_me",
		"unknown subcommand",
		"missing -config",
		"no such file",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
