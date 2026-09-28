package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxBlobSize = int64(^uint64(0) >> 1)

// LocalStore persists blobs in the legacy files/<sha256> layout.
type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if cleanRoot == "." || cleanRoot == "" {
		return nil, fmt.Errorf("blob store root cannot be empty")
	}
	if err := os.MkdirAll(cleanRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create blob store root: %w", err)
	}

	return &LocalStore{root: cleanRoot}, nil
}

func (s *LocalStore) Put(ctx context.Context, key string, body io.Reader, size int64) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	if err := validateKey(key); err != nil {
		return ObjectInfo{}, err
	}
	if body == nil {
		return ObjectInfo{}, fmt.Errorf("blob body cannot be nil")
	}
	if size < 0 {
		return ObjectInfo{}, fmt.Errorf("blob size cannot be negative")
	}
	if size == maxBlobSize {
		return ObjectInfo{}, fmt.Errorf("blob size exceeds supported limit")
	}

	temporary, err := os.CreateTemp(s.root, ".blob-*")
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("create temporary blob: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	written, copyErr := copyBlob(ctx, temporary, body, size)
	if copyErr != nil {
		_ = temporary.Close()
		return ObjectInfo{}, copyErr
	}
	if written != size {
		_ = temporary.Close()
		return ObjectInfo{}, fmt.Errorf("blob size mismatch: wrote %d bytes, expected %d", written, size)
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return ObjectInfo{}, fmt.Errorf("set blob permissions: %w", err)
	}
	closeErr := temporary.Close()
	if closeErr != nil {
		return ObjectInfo{}, fmt.Errorf("close temporary blob: %w", closeErr)
	}
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	if err := os.Rename(temporaryPath, s.pathFor(key)); err != nil {
		return ObjectInfo{}, fmt.Errorf("persist blob: %w", err)
	}

	return s.Stat(ctx, key)
}

func (s *LocalStore) Get(ctx context.Context, key string, byteRange ByteRange) (io.ReadCloser, ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, ObjectInfo{}, err
	}
	if err := validateKey(key); err != nil {
		return nil, ObjectInfo{}, err
	}

	file, err := os.Open(s.pathFor(key))
	if err != nil {
		return nil, ObjectInfo{}, mapNotFound(err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, ObjectInfo{}, fmt.Errorf("stat blob: %w", err)
	}
	objectInfo := objectInfoFromFile(key, info)
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, ObjectInfo{}, fmt.Errorf("blob %q is not a regular file", key)
	}
	if err := validateRange(byteRange, objectInfo.Size); err != nil {
		_ = file.Close()
		return nil, ObjectInfo{}, err
	}
	if byteRange.Offset == 0 && byteRange.Length == -1 {
		return file, objectInfo, nil
	}
	length := byteRange.Length
	if length == -1 {
		length = objectInfo.Size - byteRange.Offset
	}

	return &sectionReadCloser{
		Reader: io.NewSectionReader(file, byteRange.Offset, length),
		Closer: file,
	}, objectInfo, nil
}

func (s *LocalStore) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	if err := validateKey(key); err != nil {
		return ObjectInfo{}, err
	}

	info, err := os.Stat(s.pathFor(key))
	if err != nil {
		return ObjectInfo{}, mapNotFound(err)
	}
	if !info.Mode().IsRegular() {
		return ObjectInfo{}, fmt.Errorf("blob %q is not a regular file", key)
	}
	return objectInfoFromFile(key, info), nil
}

func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}
	if err := os.Remove(s.pathFor(key)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return mapNotFound(err)
	}
	return nil
}

func (s *LocalStore) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("list blobs: %w", err)
	}
	objects := make([]ObjectInfo, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.Type().IsRegular() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if err := validateKey(entry.Name()); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect blob %q: %w", entry.Name(), err)
		}
		objects = append(objects, objectInfoFromFile(entry.Name(), info))
	}

	return objects, nil
}

func (s *LocalStore) pathFor(key string) string {
	return filepath.Join(s.root, key)
}

type sectionReadCloser struct {
	io.Reader
	io.Closer
}

func objectInfoFromFile(key string, info os.FileInfo) ObjectInfo {
	return ObjectInfo{
		Key:      key,
		Size:     info.Size(),
		Modified: info.ModTime().UTC(),
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}

func validateKey(key string) error {
	if len(key) != 64 {
		return fmt.Errorf("blob key must be a 64-character SHA-256 hex digest")
	}
	for _, character := range key {
		isDigit := character >= '0' && character <= '9'
		isLowerHex := character >= 'a' && character <= 'f'
		isUpperHex := character >= 'A' && character <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return fmt.Errorf("blob key must be a 64-character SHA-256 hex digest")
		}
	}
	return nil
}

func validateRange(byteRange ByteRange, size int64) error {
	if byteRange.Offset < 0 {
		return fmt.Errorf("%w: offset cannot be negative", ErrInvalidRange)
	}
	if byteRange.Length < -1 {
		return fmt.Errorf("%w: length must be -1 or non-negative", ErrInvalidRange)
	}
	if byteRange.Offset > size {
		return fmt.Errorf("%w: offset exceeds blob size", ErrInvalidRange)
	}
	if byteRange.Length == -1 || byteRange.Length <= size-byteRange.Offset {
		return nil
	}
	return fmt.Errorf("%w: range exceeds blob size", ErrInvalidRange)
}

func copyBlob(ctx context.Context, destination io.Writer, source io.Reader, size int64) (int64, error) {
	const bufferSize = 32 * 1024

	buffer := make([]byte, bufferSize)
	maximumRead := size + 1
	var written int64

	for written < maximumRead {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		remaining := maximumRead - written
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		n, readErr := source.Read(buffer[:readSize])
		if err := ctx.Err(); err != nil {
			return written, err
		}
		if n > 0 {
			if int64(n) > size-written {
				return written, fmt.Errorf("blob body exceeds declared size %d", size)
			}
			writtenNow, writeErr := destination.Write(buffer[:n])
			written += int64(writtenNow)
			if writeErr != nil {
				return written, fmt.Errorf("write blob: %w", writeErr)
			}
			if writtenNow != n {
				return written, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, fmt.Errorf("read blob: %w", readErr)
		}
		if n == 0 {
			return written, io.ErrNoProgress
		}
	}

	return written, fmt.Errorf("blob body exceeds declared size %d", size)
}
