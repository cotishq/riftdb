package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var ErrInvalidConfig = errors.New("s3 config is incomplete")

type s3API interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// Config is an S3-compatible endpoint (MinIO locally, R2 in production).
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
}

// S3 implements Storage against an S3 API.
type S3 struct {
	api      s3API
	bucket   string
	endpoint string
}

func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	return &S3{api: client, bucket: cfg.Bucket, endpoint: cfg.Endpoint}, nil
}

func NewS3FromEnv(ctx context.Context) (*S3, error) {
	return NewS3(ctx, Config{
		Endpoint:        strings.TrimSpace(os.Getenv("R2_ENDPOINT")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		Bucket:          strings.TrimSpace(os.Getenv("R2_BUCKET_NAME")),
		Region:          strings.TrimSpace(os.Getenv("R2_REGION")),
	})
}

func (c Config) validate() error {
	if c.Endpoint == "" || c.AccessKeyID == "" || c.SecretAccessKey == "" || c.Bucket == "" {
		return ErrInvalidConfig
	}
	return nil
}

func (s *S3) Bucket() string   { return s.bucket }
func (s *S3) Endpoint() string { return s.endpoint }

func (s *S3) Ping(ctx context.Context) error {
	_, err := s.api.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return fmt.Errorf("head bucket %s: %w", s.bucket, err)
	}
	return nil
}

func (s *S3) Put(ctx context.Context, key string, data []byte) error {
	key, err := requireKey(key)
	if err != nil {
		return err
	}
	_, err = s.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	key, err := requireKey(key)
	if err != nil {
		return nil, err
	}
	out, err := s.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3) Delete(ctx context.Context, key string) error {
	key, err := requireKey(key)
	if err != nil {
		return err
	}
	_, err = s.api.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s: %w", key, err)
	}
	return nil
}

func (s *S3) List(ctx context.Context, prefix string) ([]Object, error) {
	prefix = strings.TrimSpace(prefix)
	var token *string
	out := make([]Object, 0)

	for {
		in := &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
		}
		if prefix != "" {
			in.Prefix = aws.String(prefix)
		}
		if token != nil {
			in.ContinuationToken = token
		}

		page, err := s.api.ListObjectsV2(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || *obj.Key == "" {
				continue
			}
			out = append(out, Object{Key: *obj.Key})
		}
		if !aws.ToBool(page.IsTruncated) {
			break
		}
		token = page.NextContinuationToken
		if token == nil || *token == "" {
			break
		}
	}

	return out, nil
}

func requireKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("object key is required")
	}
	return key, nil
}

func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey"
}

var _ Storage = (*S3)(nil)
var _ s3API = (*s3.Client)(nil)
