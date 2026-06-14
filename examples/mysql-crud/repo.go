package main

import (
	"context"
	"errors"
	"log"

	"gorm.io/gorm"
)

// UserRepo wraps a *gorm.DB and exposes 5 standard CRUD operations.
// All methods are context-aware so callers can pass timeouts and have
// their context propagate to the underlying SQL driver and the OTel
// tracer registered on the global TracerProvider.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo constructs a UserRepo bound to the given *gorm.DB.
// Callers should pass the *gorm.DB returned by one of the four
// dbsql.Open* helpers; this keeps repo construction orthogonal to
// tracing setup.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// CreateUser inserts a new row. The created user is reflected back via
// u.ID after the call (GORM populates the primary key).
func (r *UserRepo) CreateUser(ctx context.Context, u *User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return err
	}
	return nil
}

// GetUser fetches a single user by primary key.
// Returns gorm.ErrRecordNotFound when no row matches; callers may use
// errors.Is to distinguish from driver-level failures.
func (r *UserRepo) GetUser(ctx context.Context, id uint) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers returns up to size users. A non-positive size is corrected
// to 10 to keep the call site simple.
func (r *UserRepo) ListUsers(ctx context.Context, size int) ([]User, error) {
	if size <= 0 {
		size = 10
	}
	var out []User
	if err := r.db.WithContext(ctx).Limit(size).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateUser updates the row identified by id using a *User as the
// payload source. RowsAffected == 0 is treated as a soft failure:
// the row either does not exist or the payload is identical to the
// stored value. Both cases are not errors worth bubbling up.
func (r *UserRepo) UpdateUser(ctx context.Context, id uint, u *User) error {
	res := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(u)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		log.Printf("[WARN] update: id=%d not affected (row missing or no change)", id)
	}
	return nil
}

// DeleteUser hard-deletes the row identified by id.
// Soft-delete is intentionally not used here to keep the example minimal.
func (r *UserRepo) DeleteUser(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&User{}, id).Error; err != nil {
		return err
	}
	return nil
}

// IsNotFound is a thin wrapper around errors.Is that lets the caller
// stay decoupled from gorm.ErrRecordNotFound without an extra import.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
