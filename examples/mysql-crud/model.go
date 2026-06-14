package main

import "time"

// User is a minimal domain entity used by the example.
// Schema is intentionally simple so the focus stays on the dbx wiring.
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Age       int       `gorm:"not null;default:0" json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName pins the table to "users" so the example does not depend on
// GORM's pluralization rules.
func (User) TableName() string {
	return "users"
}
