package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Store persists blobs in an existing S3-compatible bucket.
type S3Store struct {
	client    *minio.Client
	bucket    string
	keyPrefix string
}

// NewS3Store connects to an existing S3-compatible bucket. It never creates
// buckets, so a missing bucket or insufficient credentials fail construction.
func NewS3Store(ctx context.Context, cfg config.StoreS3Config) (*S3Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	endpoint, secure, err := parseS3Endpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket cannot be empty")
	}
	accessKey := strings.TrimSpace(cfg.AccessKey)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3 access key and secret key cannot be empty")
	}

	bucketLookup := minio.BucketLookupDNS
	if cfg.UsePathStyle {
		bucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       secure,
		Region:       strings.TrimSpace(cfg.Region),
		BucketLookup: bucketLookup,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("verify S3 bucket: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("S3 bucket %q does not exist", bucket)
	}

	return &S3Store{
		client:    client,
		bucket:    bucket,
		keyPrefix: normalizeS3KeyPrefix(cfg.KeyPrefix),
	}, nil
}

func (s *S3Store) Put(ctx context.Context, key string, body io.Reader, size int64) (ObjectInfo, error) {
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
	if existing, err := s.Stat(ctx, key); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return ObjectInfo{}, err
	}

	temporary, err := os.CreateTemp("", "nrserver-s3-*")
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("create temporary S3 blob: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	written, copyErr := copyBlob(ctx, temporary, body, size)
	closeErr := temporary.Close()
	if copyErr != nil {
		return ObjectInfo{}, copyErr
	}
	if closeErr != nil {
		return ObjectInfo{}, fmt.Errorf("close temporary S3 blob: %w", closeErr)
	}
	if written != size {
		return ObjectInfo{}, fmt.Errorf("blob size mismatch: wrote %d bytes, expected %d", written, size)
	}
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	temporary, err = os.Open(temporaryPath)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("open temporary S3 blob: %w", err)
	}
	defer temporary.Close()

	objectName := s.objectName(key)
	_, err = s.client.PutObject(ctx, s.bucket, objectName, temporary, size, minio.PutObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("put S3 blob: %w", mapS3Error(err))
	}

	return s.Stat(ctx, key)
}

func (s *S3Store) Get(ctx context.Context, key string, byteRange ByteRange) (io.ReadCloser, ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, ObjectInfo{}, err
	}
	if err := validateKey(key); err != nil {
		return nil, ObjectInfo{}, err
	}

	info, err := s.Stat(ctx, key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	if err := validateRange(byteRange, info.Size); err != nil {
		return nil, ObjectInfo{}, err
	}
	if byteRange.Length == 0 {
		return io.NopCloser(strings.NewReader("")), info, nil
	}

	options := minio.GetObjectOptions{}
	if byteRange.Offset != 0 || byteRange.Length != -1 {
		end := int64(0)
		if byteRange.Length == -1 {
			end = 0
		} else {
			end = byteRange.Offset + byteRange.Length - 1
		}
		if err := options.SetRange(byteRange.Offset, end); err != nil {
			return nil, ObjectInfo{}, fmt.Errorf("set S3 blob range: %w", err)
		}
	}

	reader, err := s.client.GetObject(ctx, s.bucket, s.objectName(key), options)
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get S3 blob: %w", mapS3Error(err))
	}
	return reader, info, nil
}

func (s *S3Store) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	if err := validateKey(key); err != nil {
		return ObjectInfo{}, err
	}

	info, err := s.client.StatObject(ctx, s.bucket, s.objectName(key), minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat S3 blob: %w", mapS3Error(err))
	}
	return objectInfoFromS3(key, info), nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}

	err := s.client.RemoveObject(ctx, s.bucket, s.objectName(key), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete S3 blob: %w", mapS3Error(err))
	}
	return nil
}

func (s *S3Store) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	objects := []ObjectInfo{}
	for item := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    s.keyPrefix + prefix,
		Recursive: true,
	}) {
		if item.Err != nil {
			return nil, fmt.Errorf("list S3 blobs: %w", mapS3Error(item.Err))
		}
		key, ok := s.keyForObject(item.Key)
		if !ok || !strings.HasPrefix(key, prefix) {
			continue
		}
		objects = append(objects, objectInfoFromS3(key, item))
	}
	return objects, nil
}

// PresignedGet returns a temporary, signed URL for an existing blob.
func (s *S3Store) PresignedGet(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("S3 presign TTL must be greater than zero")
	}
	if _, err := s.Stat(ctx, key); err != nil {
		return nil, err
	}

	link, err := s.client.PresignedGetObject(ctx, s.bucket, s.objectName(key), ttl, nil)
	if err != nil {
		return nil, fmt.Errorf("presign S3 blob GET: %w", mapS3Error(err))
	}
	return link, nil
}

func (s *S3Store) objectName(key string) string {
	return s.keyPrefix + key
}

func (s *S3Store) keyForObject(objectName string) (string, bool) {
	key, ok := strings.CutPrefix(objectName, s.keyPrefix)
	if !ok || validateKey(key) != nil {
		return "", false
	}
	return key, true
}

func parseS3Endpoint(value string) (string, bool, error) {
	endpointURL, err := url.Parse(strings.TrimSpace(value))
	if err != nil || endpointURL.Host == "" {
		return "", false, fmt.Errorf("S3 endpoint must be an absolute HTTP or HTTPS URL")
	}
	if endpointURL.User != nil || (endpointURL.Path != "" && endpointURL.Path != "/") || endpointURL.RawQuery != "" || endpointURL.Fragment != "" {
		return "", false, fmt.Errorf("S3 endpoint must not include credentials, a path, query, or fragment")
	}
	switch endpointURL.Scheme {
	case "http":
		return endpointURL.Host, false, nil
	case "https":
		return endpointURL.Host, true, nil
	default:
		return "", false, fmt.Errorf("S3 endpoint scheme must be http or https")
	}
}

func normalizeS3KeyPrefix(prefix string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func objectInfoFromS3(key string, info minio.ObjectInfo) ObjectInfo {
	return ObjectInfo{
		Key:      key,
		Size:     info.Size,
		Modified: info.LastModified.UTC(),
	}
}

func mapS3Error(err error) error {
	if err == nil {
		return nil
	}
	response := minio.ToErrorResponse(err)
	if response.Code == minio.NoSuchKey || response.Code == minio.NoSuchBucket || response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	return err
}

var _ Store = (*S3Store)(nil)
