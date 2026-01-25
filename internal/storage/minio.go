package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage struct {
	client   *minio.Client
	bucket   string
	endpoint string
}

// NewFileStorage 初始化 MinIO 连接
func NewFileStorage(endpoint, accessKey, secretKey, bucketName string) *FileStorage {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // 本地开发通常用 HTTP (false), 生产环境用 HTTPS (true)
	})
	if err != nil {
		log.Fatalln(err)
	}

	// 自动创建 Bucket (如果不存在)
	// 实际生产中建议手动创建，或者在这里加个 Check
	ctx := context.Background()
	exists, _ := minioClient.BucketExists(ctx, bucketName)
	if !exists {
		minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		// 设置为公开访问策略，否则图片无法直接访问
		policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucketName)
		minioClient.SetBucketPolicy(ctx, bucketName, policy)
	}

	return &FileStorage{
		client:   minioClient,
		bucket:   bucketName,
		endpoint: endpoint,
	}
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

	// 拼接访问 URL
	// 本地开发: http://localhost:9000/notes-images/xxxx.jpg
	fileURL := fmt.Sprintf("http://%s/%s/%s", s.endpoint, s.bucket, fileName)
	return fileURL, nil
}
