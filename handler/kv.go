package handler

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")

	ErrAlreadyExists = errors.New("already exists")
)

type KVStore interface {
	Get(ctx context.Context, key []byte) ([]byte, error)

	Put(ctx context.Context, key []byte, value []byte) error

	Delete(ctx context.Context, key []byte) error
}
