package providers

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Provider implements the Provider interface for AWS S3 and S3-compatible services
type S3Provider struct {
	client *s3.S3
	bucket string
}

// NewS3Provider creates a new S3 provider
func NewS3Provider(opts ProviderOptions) (*S3Provider, error) {
	config := &aws.Config{
		Region: aws.String(opts.Region),
	}

	// Set custom endpoint for S3-compatible services (like DigitalOcean Spaces)
	if opts.Endpoint != "" {
		config.Endpoint = aws.String(opts.Endpoint)
		config.S3ForcePathStyle = aws.Bool(true)
	}

	// Set credentials if provided
	if opts.AccessKey != "" && opts.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(
			opts.AccessKey,
			opts.SecretKey,
			"",
		)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Provider{
		client: s3.New(sess),
		bucket: opts.Bucket,
	}, nil
}

// ListObjects lists all objects with the given prefix
func (p *S3Provider) ListObjects(ctx context.Context, prefix string) ([]Object, error) {
	var objects []Object
	
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucket),
		Prefix: aws.String(prefix),
	}

	err := p.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			objects = append(objects, Object{
				Key:          aws.StringValue(obj.Key),
				Size:         aws.Int64Value(obj.Size),
				LastModified: aws.TimeValue(obj.LastModified),
				ETag:         aws.StringValue(obj.ETag),
			})
		}
		return !lastPage
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return objects, nil
}

// DownloadObject downloads a specific object
func (p *S3Provider) DownloadObject(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	result, err := p.client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download object %s: %w", key, err)
	}

	return result.Body, nil
}

// GetObjectInfo gets metadata about an object
func (p *S3Provider) GetObjectInfo(ctx context.Context, key string) (*Object, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	result, err := p.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info for %s: %w", key, err)
	}

	metadata := make(map[string]string)
	for k, v := range result.Metadata {
		metadata[k] = aws.StringValue(v)
	}

	return &Object{
		Key:          key,
		Size:         aws.Int64Value(result.ContentLength),
		LastModified: aws.TimeValue(result.LastModified),
		ETag:         aws.StringValue(result.ETag),
		ContentType:  aws.StringValue(result.ContentType),
		Metadata:     metadata,
	}, nil
}

// Close cleans up any resources used by the provider
func (p *S3Provider) Close() error {
	// S3 client doesn't need explicit cleanup
	return nil
} 