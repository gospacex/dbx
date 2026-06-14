package main

import (
	"errors"

	"gorm.io/gorm"
)

// isRecordNotFound reports whether err is GORM's not-found sentinel.
// Kept in its own file so logger.go stays focused on the gorm/logger.Interface.
func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
