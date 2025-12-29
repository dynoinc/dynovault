package handler

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrConditionFailed = errors.New("condition failed")
)

// Version represents an item version for optimistic concurrency control.
// Version 0 indicates the item does not exist.
type Version uint64

// Result holds the value and metadata from a Get operation.
type Result struct {
	Value   []byte
	Version Version
}

// PutOption configures a conditional Put operation.
type PutOption func(*PutConfig)

// PutConfig holds configuration for a Put operation.
type PutConfig struct {
	ExpectedVersion *Version
}

// IfNotExists makes Put succeed only if the key doesn't exist.
// Semantically equivalent to IfVersion(0).
func IfNotExists() PutOption {
	return func(c *PutConfig) {
		v := Version(0)
		c.ExpectedVersion = &v
	}
}

// IfVersion makes Put succeed only if the current version matches.
// Version 0 means the key must not exist.
func IfVersion(v Version) PutOption {
	return func(c *PutConfig) {
		c.ExpectedVersion = &v
	}
}

// KVStore defines a key-value store with optimistic concurrency control.
type KVStore interface {
	// Get retrieves a value by key. Returns ErrNotFound if the key doesn't exist.
	Get(ctx context.Context, key []byte) (*Result, error)

	// Put stores a value. Optional conditions make it atomic.
	// Returns ErrConditionFailed if a condition is not met.
	Put(ctx context.Context, key []byte, value []byte, opts ...PutOption) error

	// Delete removes a key.
	Delete(ctx context.Context, key []byte) error
}
