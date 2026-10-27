package file

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"wfagent/internal/base/config"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg config.FileConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessID, cfg.MinIOSecret, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOClient{client: client, bucket: cfg.MinIOBucket}, nil
}

func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
}

func (m *MinIOClient) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	if err := m.EnsureBucket(ctx); err != nil {
		return "", err
	}
	_, err := m.client.PutObject(ctx, m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return objectName, nil
}

func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	values := make(url.Values)
	u, err := m.client.PresignedGetObject(ctx, m.bucket, objectName, expiry, values)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *MinIOClient) Download(ctx context.Context, objectName string) (*minio.Object, error) {
	return m.client.GetObject(ctx, m.bucket, objectName, minio.GetObjectOptions{})
}

func (m *MinIOClient) Delete(ctx context.Context, objectName string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
}
