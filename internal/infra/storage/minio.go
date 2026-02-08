package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	publicURL string
}

// NewFileStorage 初始化 MinIO 连接
func NewFileStorage(endpoint, publicURL, accessKey, secretKey, bucketName string) (*FileStorage, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // 本地开发通常用 HTTP (false), 生产环境用 HTTPS (true)
	})
	if err != nil {
		return nil, err
	}

	// 自动创建 Bucket (如果不存在)
	// 实际生产中建议手动创建，或者在这里加个 Check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, errBucket := minioClient.BucketExists(ctx, bucketName)
	if errBucket == nil && !exists {
		err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err == nil {
			// 只有创建成功才设置策略
			policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucketName)
			_ = minioClient.SetBucketPolicy(ctx, bucketName, policy)
			log.Printf("Bucket %s created and policy set.", bucketName)
		} else {
			// 记录错误但不 Panic，可能只是权限不足，但 Bucket 已经存在
			log.Printf("Failed to create bucket: %v", err)
		}
	}

	return &FileStorage{
		client:    minioClient,
		bucket:    bucketName,
		endpoint:  endpoint,
		publicURL: publicURL,
	}, nil
}

// UploadImage 上传图片并返回 URL
// fileData: 图片文件的二进制流
// fileName: 文件名 (建议用 UUID 生成唯一文件名)
// contentType: 例如 "image/jpeg"
func (s *FileStorage) UploadImage(ctx context.Context, fileName string, fileSize int64, reader io.Reader, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, fileName, reader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	// 拼接 URL 时的细节处理：
	// 如果 publicURL 是 "http://localhost:9000/"，最后会变成 "//bucket"
	// 所以要先 TrimRight
	baseURL := strings.TrimRight(s.publicURL, "/")
	// 注意：这里不用 path.Join，因为它会把 http:// 变成 http:/
	// 手动拼接是最稳的
	fileURL := fmt.Sprintf("%s/%s/%s", baseURL, s.bucket, fileName)
	return fileURL, nil
}
