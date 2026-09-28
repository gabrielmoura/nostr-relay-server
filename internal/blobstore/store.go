// Package blobstore defines storage primitives for Blossom blob content.
package blobstore

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound     = errors.New("blob not found")
	ErrInvalidRange = errors.New("invalid blob range")
)

// ByteRange selects a contiguous region of a blob. A Length of -1 reads from
// Offset through the end of the blob; zero is a valid empty range.
type ByteRange struct {
	Offset int64
	Length int64
}

// ObjectInfo describes a blob stored under its SHA-256 key.
type ObjectInfo struct {
	Key      string
	Size     int64
	Modified time.Time
}

// Store is the backend-neutral contract for blob persistence.
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64) (ObjectInfo, error)
	Get(ctx context.Context, key string, byteRange ByteRange) (io.ReadCloser, ObjectInfo, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
}
