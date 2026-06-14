// Package dbx provides GORM access toolkit with unified config and trace export.
package dbx

import "strings"

var (
	// ErrInvalidConfig is returned when a database configuration is invalid.
	ErrInvalidConfig = &sentinelError{"dbx: invalid config"}
	// ErrDriverUnsupported is returned when a driver is not supported.
	ErrDriverUnsupported = &sentinelError{"dbx: unsupported driver"}
)

type sentinelError struct{ msg string }

func (e *sentinelError) Error() string { return e.msg }
func (e *sentinelError) Is(target error) bool {
	t, ok := target.(*sentinelError)
	return ok && e.msg == t.msg
}

// RedactDSN replaces password portion in a DSN string for safe logging.
func RedactDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	colonIdx := strings.IndexByte(dsn, ':')
	atIdx := strings.IndexByte(dsn, '@')
	if colonIdx >= 0 && atIdx > colonIdx {
		return dsn[:colonIdx+1] + "xxxxx" + dsn[atIdx:]
	}
	return dsn
}
