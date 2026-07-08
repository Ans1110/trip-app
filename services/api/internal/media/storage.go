package media

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStat is what the Media Service trusts. It comes from HeadObject —
// never from the client. mime / bytes / etag on the final media.assets row
// are always these values, not what the InitUpload request declared.
type ObjectStat struct {
	Size int64
	Mime string
	ETag string
}

type Storage interface {
	EnsureBucket(ctx context.Context) error
	PresignPut(ctx context.Context, key, mime string, ttl time.Duration) (*url.URL, error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (*url.URL, error)
	StatObject(ctx context.Context, key string) (*ObjectStat, error)
	RemoveObject(ctx context.Context, key string) error
	Bucket() string
	// GenerateKey produces a server-side object key. Client-supplied names are
	// never used — this is what stops path traversal and cross-purpose writes.
	GenerateKey(prefix, ext string) string
}

var (
	ErrObjectNotFound = errors.New("media: object not found")
)

type minioStorage struct {
	client         *minio.Client
	bucket         string
	presignBaseURL *url.URL // used to rewrite Endpoint→PublicEndpoint on presigned URLs
	publicHost     string
	publicScheme   string
}

func NewStorage(cfg config.MediaConfig) (Storage, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("media: endpoint and bucket are required")
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("media: minio.New: %w", err)
	}

	s := &minioStorage{client: client, bucket: cfg.Bucket}
	if cfg.PublicEndpoint != "" {
		u, err := parseEndpoint(cfg.PublicEndpoint, cfg.UseSSL)
		if err != nil {
			return nil, fmt.Errorf("media: parse public_endpoint: %w", err)
		}
		s.publicHost = u.Host
		s.publicScheme = u.Scheme
	}
	return s, nil
}

// parseEndpoint accepts either "host:port" or a full "https://host" style
// value. Bare hosts pick their scheme from useSSL.
func parseEndpoint(raw string, useSSL bool) (*url.URL, error) {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	if !hasScheme(raw) {
		raw = scheme + "://" + raw
	}
	return url.Parse(raw)
}

func hasScheme(raw string) bool {
	for i, r := range raw {
		if r == ':' && i > 0 {
			return true
		}
		if r == '/' || r == '?' || r == '#' {
			return false
		}
	}
	return false
}

func (s *minioStorage) Bucket() string { return s.bucket }

// EnsureBucket is idempotent — safe to call on boot every time.
func (s *minioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("media: bucket exists: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("media: make bucket: %w", err)
	}
	return nil
}

func (s *minioStorage) PresignPut(ctx context.Context, key, mime string, ttl time.Duration) (*url.URL, error) {
	// Presign a plain PUT. The client must send Content-Type matching `mime` when uploading
	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return nil, fmt.Errorf("media: presign put: %w", err)
	}
	return s.rewriteHost(u), nil
}

func (s *minioStorage) PresignGet(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("media: presign get: %w", err)
	}
	return s.rewriteHost(u), nil
}

func (s *minioStorage) StatObject(ctx context.Context, key string) (*ObjectStat, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotExist(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("media: stat: %w", err)
	}
	return &ObjectStat{
		Size: info.Size,
		Mime: info.ContentType,
		ETag: info.ETag,
	}, nil
}

func (s *minioStorage) RemoveObject(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil && !isNotExist(err) {
		return fmt.Errorf("media: remove: %w", err)
	}
	return nil
}

// GenerateKey lays out objects as {prefix}{yyyy}/{mm}/{uuid}{ext}
func (s *minioStorage) GenerateKey(prefix, ext string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s%04d/%02d/%s%s",
		prefix, now.Year(), int(now.Month()), uuid.NewString(), ext)
}

// rewriteHost swaps the SDK's server host with the caller-visible host so
// the presigned URL is usable from the browser. Signature stays valid because
// the signature covers the path + params, not the authority.
func (s *minioStorage) rewriteHost(u *url.URL) *url.URL {
	if s.publicHost == "" {
		return u
	}
	rewritten := *u
	rewritten.Host = s.publicHost
	if s.publicScheme != "" {
		rewritten.Scheme = s.publicScheme
	}
	return &rewritten
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	var resp minio.ErrorResponse
	if errors.As(err, &resp) {
		return resp.StatusCode == http.StatusNotFound || resp.Code == "NoSuchKey"
	}
	return false
}
