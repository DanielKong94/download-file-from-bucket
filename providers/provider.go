package providers

import (
	"context"
	"io"
	"time"
)

// Provider defines the interface that all cloud storage providers must implement
type Provider interface {
	// ListObjects lists all objects with the given prefix
	ListObjects(ctx context.Context, prefix string) ([]Object, error)
	
	// DownloadObject downloads a specific object
	DownloadObject(ctx context.Context, key string) (io.ReadCloser, error)
	
	// GetObjectInfo gets metadata about an object
	GetObjectInfo(ctx context.Context, key string) (*Object, error)
	
	// Close cleans up any resources used by the provider
	Close() error
}

// Object represents a cloud storage object
type Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type"`
	Metadata     map[string]string `json:"metadata"`
}

// DownloadProgress represents the progress of a download operation
type DownloadProgress struct {
	Key           string
	BytesDownloaded int64
	TotalBytes    int64
	Error         error
	Completed     bool
}

// ProviderType represents the type of cloud storage provider
type ProviderType string

const (
	ProviderTypeS3           ProviderType = "s3"
	ProviderTypeDigitalOcean ProviderType = "digitalocean"
)

// ProviderOptions holds configuration options for creating providers
type ProviderOptions struct {
	Type      ProviderType
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Options   map[string]string
} 