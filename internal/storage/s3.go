// Package storage provides an S3-compatible object store abstraction.
// Works with any S3-compatible provider: AWS, Garage, Hetzner Object Storage,
// Contabo Object Storage, Cloudflare R2, MinIO, etc.
// Multi-provider failover: if the primary upload fails, it retries on secondary
// providers in order until one succeeds.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/google/uuid"
)

// Store wraps an S3 client for a specific bucket / provider.
type Store struct {
	client   *s3.Client
	bucket   string
	provider string
}

// New creates a Store from config. Works with any S3-compatible endpoint.
func New(ctx context.Context, cfg config.S3Config, provider string) (*Store, error) {
	opts := s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
	}
	if cfg.Endpoint != "" {
		opts.BaseEndpoint = aws.String(cfg.Endpoint)
	}
	client := s3.New(opts)

	store := &Store{client: client, bucket: cfg.Bucket, provider: provider}

	if err := store.ensureBucketExists(ctx); err != nil {
		return nil, fmt.Errorf("storage: ensure bucket exists: %w", err)
	}

	return store, nil
}

// ensureBucketExists checks if the bucket exists and creates it if it doesn't.
func (s *Store) ensureBucketExists(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	// Check if it is a 404 (Not Found)
	// The aws-sdk-go-v2 error handling for S3 HeadBucket 404 is a bit specific.
	// It usually returns a generic NotFound error or 404 status code.
	// For simplicity, we try to create it if HeadBucket failed.
	// A more robust check would inspect the error type.

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		// If CreateBucket fails, it might be because it already exists (owned by someone else)
		// or insufficient permissions.
		// However, for local zero-config with Garage, this should work if creds are correct.
		return fmt.Errorf("failed to create bucket %s: %w", s.bucket, err)
	}

	return nil
}

// Provider returns the human-readable provider label.
func (s *Store) Provider() string { return s.provider }

// PutLogs compresses raw NDJSON bytes and uploads to S3.
// Returns: S3 key, SHA-256 hex of compressed bytes, compressed byte count, log line count.
// Uses a deterministic key so duplicate uploads are idempotent.
func (s *Store) PutLogs(ctx context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte) (key, sha256hex string, compressedBytes, logLines int64, err error) {
	compressed, meta, err := PrepareBlob(raw, customerID, zoneID, from, to)
	if err != nil {
		return "", "", 0, 0, err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(meta.Key),
		Body:          bytes.NewReader(compressed),
		ContentLength: aws.Int64(meta.Size),
		ContentType:   aws.String("application/x-ndjson+gzip"),
		Metadata:      map[string]string{"sha256": meta.SHA256},
	})
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("storage: put object: %w", err)
	}
	return meta.Key, meta.SHA256, meta.Size, meta.Lines, nil
}

// GetLogs downloads and decompresses a stored log object.
func (s *Store) GetLogs(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: get object: %w", err)
	}
	defer out.Body.Close()

	return DecompressBlob(out.Body)
}

// DeleteObject removes an object (used by GDPR art.17 expiry worker).
func (s *Store) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: delete: %w", err)
	}
	return nil
}

// ── Multi-provider failover ───────────────────────────────────────────────────

// MultiStore tries providers in order and returns on first success.
type MultiStore struct {
	providers []Backend
}

// NewMultiStore creates a MultiStore from a list of Stores (primary first).
func NewMultiStore(providers ...Backend) *MultiStore {
	return &MultiStore{providers: providers}
}

// PutLogs uploads to the first available provider.
// Returns the winning provider label alongside the object metadata.
func (m *MultiStore) PutLogs(ctx context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte) (key, sha256hex, provider string, compressedBytes, logLines int64, err error) {
	for _, p := range m.providers {
		var k, h string
		var cb, ll int64
		k, h, cb, ll, err = p.PutLogs(ctx, customerID, zoneID, from, to, raw)
		if err == nil {
			return k, h, p.Provider(), cb, ll, nil
		}
	}
	return "", "", "", 0, 0, fmt.Errorf("storage: all providers failed, last error: %w", err)
}

// GetLogs fetches from the first provider that has the object.
func (m *MultiStore) GetLogs(ctx context.Context, key string) ([]byte, error) {
	var lastErr error
	for _, p := range m.providers {
		data, err := p.GetLogs(ctx, key)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("storage: all providers failed: %w", lastErr)
}

// DeleteObject deletes from all providers (best-effort).
func (m *MultiStore) DeleteObject(ctx context.Context, key string) error {
	var lastErr error
	for _, p := range m.providers {
		if err := p.DeleteObject(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
