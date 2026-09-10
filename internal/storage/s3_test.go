package storage

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type stubAPI struct {
	objects map[string][]byte
	pages   int
}

func newStub() *stubAPI {
	return &stubAPI{objects: make(map[string][]byte)}
}

func (s *stubAPI) HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}

func (s *stubAPI) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	s.objects[*in.Key] = data
	return &s3.PutObjectOutput{}, nil
}

func (s *stubAPI) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := s.objects[*in.Key]
	if !ok {
		return nil, &types.NoSuchKey{Message: aws.String("not found")}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (s *stubAPI) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	delete(s.objects, *in.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func (s *stubAPI) ListObjectsV2(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	prefix := ""
	if in.Prefix != nil {
		prefix = *in.Prefix
	}

	keys := make([]string, 0)
	for key := range s.objects {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	pageSize := len(keys)
	if s.pages > 0 && s.pages < pageSize {
		pageSize = s.pages
	}

	start := 0
	if in.ContinuationToken != nil && *in.ContinuationToken != "" {
		for i, key := range keys {
			if key == *in.ContinuationToken {
				start = i
				break
			}
		}
	}

	end := start + pageSize
	if end > len(keys) {
		end = len(keys)
	}

	contents := make([]types.Object, 0, end-start)
	for _, key := range keys[start:end] {
		k := key
		contents = append(contents, types.Object{Key: &k})
	}

	out := &s3.ListObjectsV2Output{Contents: contents}
	if end < len(keys) {
		out.IsTruncated = aws.Bool(true)
		out.NextContinuationToken = aws.String(keys[end])
	}
	return out, nil
}

func TestS3PutGetList(t *testing.T) {
	ctx := context.Background()
	api := newStub()
	store := &S3{api: api, bucket: "riftdb"}

	if err := store.Put(ctx, "collections/papers/notes.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "collections/other/x.txt", []byte("nope")); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ctx, "collections/papers/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}

	listed, err := store.List(ctx, "collections/papers")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Key != "collections/papers/notes.txt" {
		t.Fatalf("list: %+v", listed)
	}
}

func TestS3GetMissing(t *testing.T) {
	store := &S3{api: newStub(), bucket: "riftdb"}
	_, err := store.Get(context.Background(), "missing.txt")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestS3ListPages(t *testing.T) {
	ctx := context.Background()
	api := newStub()
	api.pages = 1
	store := &S3{api: api, bucket: "riftdb"}

	if err := store.Put(ctx, "a.txt", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "b.txt", []byte("b")); err != nil {
		t.Fatal(err)
	}

	listed, err := store.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("paged list: got %d, want 2", len(listed))
	}
}

func TestNewS3FromEnvRequiresConfig(t *testing.T) {
	t.Setenv("R2_ENDPOINT", "")
	t.Setenv("R2_ACCESS_KEY_ID", "")
	t.Setenv("R2_SECRET_ACCESS_KEY", "")
	t.Setenv("R2_BUCKET_NAME", "")

	_, err := NewS3FromEnv(context.Background())
	if err != ErrInvalidConfig {
		t.Fatalf("got %v, want ErrInvalidConfig", err)
	}
}
