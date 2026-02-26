package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Default QStorage configuration
const (
	DefaultRegion      = "q-world-1"
	DefaultEndpointURL = "https://qstorage.quilibrium.com"
)

// Standard bucket layout directories (mirrors local .config/ structure)
const (
	BucketDirStore       = "store"        // master store data
	BucketDirWorkerStore = "worker-store" // worker-store/<id>/...
)

// ClientConfig holds the configuration for creating an S3 client
type ClientConfig struct {
	AccessKeyID string
	AccessKey   string
	AccountID   string
	Region      string
	EndpointURL string
	Bucket      string
	Prefix      string // Optional root prefix within the bucket
}

// Client wraps the AWS S3 client with QStorage defaults
type Client struct {
	s3Client *s3.Client
	config   ClientConfig
}

// NewClient creates a new S3 client configured for QStorage (or custom S3-compatible endpoint)
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("access key ID is required")
	}
	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("access key is required")
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	// Apply defaults
	if cfg.Region == "" {
		cfg.Region = DefaultRegion
	}
	if cfg.EndpointURL == "" {
		cfg.EndpointURL = DefaultEndpointURL
	}

	// Create AWS config with static credentials and custom endpoint
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.EndpointURL)
		o.UsePathStyle = true // Required for most S3-compatible services
	})

	return &Client{
		s3Client: s3Client,
		config:   cfg,
	}, nil
}

// DownloadFile downloads a file from the configured S3 bucket to a local path
func (c *Client) DownloadFile(ctx context.Context, key string, destPath string) error {
	// Ensure destination directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get the object from S3
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download %s from bucket %s: %w", key, c.config.Bucket, err)
	}
	defer output.Body.Close()

	// Create the destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	// Copy the content
	if _, err := io.Copy(file, output.Body); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

// UploadFile uploads a local file to the configured S3 bucket
func (c *Client) UploadFile(ctx context.Context, localPath string, key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to bucket %s: %w", key, c.config.Bucket, err)
	}

	return nil
}

// ListObjects lists all objects in the configured bucket with an optional prefix.
// Handles pagination automatically to return all results.
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.config.Bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	for {
		output, err := c.s3Client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in bucket %s: %w", c.config.Bucket, err)
		}

		for _, obj := range output.Contents {
			keys = append(keys, *obj.Key)
		}

		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
		input.ContinuationToken = output.NextContinuationToken
	}

	return keys, nil
}

// UploadDirectory recursively uploads all files from a local directory to the bucket
// under the given S3 key prefix. Preserves relative directory structure.
func (c *Client) UploadDirectory(ctx context.Context, localDir string, s3Prefix string) (int, error) {
	count := 0

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Build relative path from localDir
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}

		// Convert OS path separators to forward slashes for S3 keys
		s3Key := s3Prefix + "/" + filepath.ToSlash(relPath)

		if err := c.UploadFile(ctx, path, s3Key); err != nil {
			return err
		}
		count++
		return nil
	})

	return count, err
}

// DownloadDirectory downloads all objects under an S3 prefix to a local directory.
// Preserves the relative key structure as subdirectories.
func (c *Client) DownloadDirectory(ctx context.Context, s3Prefix string, localDir string) (int, error) {
	keys, err := c.ListObjects(ctx, s3Prefix)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, key := range keys {
		// Strip the prefix to get relative path
		relKey := key
		if s3Prefix != "" && len(key) > len(s3Prefix) {
			relKey = key[len(s3Prefix):]
			// Trim leading slash
			if len(relKey) > 0 && relKey[0] == '/' {
				relKey = relKey[1:]
			}
		}

		if relKey == "" {
			continue // Skip the prefix "directory" itself
		}

		destPath := filepath.Join(localDir, filepath.FromSlash(relKey))

		if err := c.DownloadFile(ctx, key, destPath); err != nil {
			return count, fmt.Errorf("failed to download %s: %w", key, err)
		}
		count++
	}

	return count, nil
}

// DeleteObject deletes an object from the configured bucket
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s from bucket %s: %w", key, c.config.Bucket, err)
	}

	return nil
}

// GetBucket returns the configured bucket name
func (c *Client) GetBucket() string {
	return c.config.Bucket
}

// GetPrefix returns the configured prefix
func (c *Client) GetPrefix() string {
	return c.config.Prefix
}

// GetObjectKey builds a full S3 key by joining the optional prefix with path segments.
// Example: prefix="mynode", segments=["config", "keys.yml"] -> "mynode/config/keys.yml"
// Example: prefix="", segments=["config", "keys.yml"] -> "config/keys.yml"
func (c *Client) GetObjectKey(segments ...string) string {
	parts := []string{}
	if c.config.Prefix != "" {
		parts = append(parts, c.config.Prefix)
	}
	parts = append(parts, segments...)

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "/"
		}
		result += p
	}
	return result
}
