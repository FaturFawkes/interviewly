package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/domain"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioResumeStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOResumeStorage(cfg *config.Config) (domain.ResumeFileStorage, error) {
	endpoint := strings.TrimSpace(cfg.MinIOEndpoint)
	accessKey := strings.TrimSpace(cfg.MinIOAccessKey)
	secretKey := strings.TrimSpace(cfg.MinIOSecretKey)
	bucket := strings.TrimSpace(cfg.MinIOBucket)
	region := strings.TrimSpace(cfg.MinIORegion)

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, nil
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: cfg.MinIOUseSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize minio client: %w", err)
	}

	storage := &minioResumeStorage{
		client: client,
		bucket: bucket,
	}

	if err := storage.ensureBucket(region); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *minioResumeStorage) UploadResume(userID, fileName, contentType string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("resume file is empty")
	}

	userSegment := normalizePathSegment(userID)
	if userSegment == "" {
		userSegment = "anonymous"
	}

	extension := fileExtension(fileName, contentType)
	objectKey := fmt.Sprintf(
		"resumes/%s/%s-%s%s",
		userSegment,
		time.Now().UTC().Format("20060102T150405"),
		uuid.NewString(),
		extension,
	)

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	userMetadata := map[string]string{}
	if strings.TrimSpace(fileName) != "" {
		userMetadata["original-filename"] = filepath.Base(strings.TrimSpace(fileName))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata},
	)
	if err != nil {
		return "", fmt.Errorf("upload resume to minio: %w", err)
	}

	return fmt.Sprintf("minio://%s/%s", s.bucket, objectKey), nil
}

func (s *minioResumeStorage) DownloadResume(minIOPath string) (*domain.ResumeFile, error) {
	bucket, objectKey, err := parseMinIOPath(minIOPath, s.bucket)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	object, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get resume object from minio: %w", err)
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat resume object from minio: %w", err)
	}

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read resume object from minio: %w", err)
	}

	fileName := resolveOriginalFileName(info.UserMetadata, objectKey)
	contentType := strings.TrimSpace(info.ContentType)
	if contentType == "" {
		contentType = fallbackContentType(fileName)
	}

	return &domain.ResumeFile{
		FileName:    fileName,
		ContentType: contentType,
		Data:        data,
	}, nil
}

func (s *minioResumeStorage) DeleteResume(minIOPath string) error {
	bucket, objectKey, err := parseMinIOPath(minIOPath, s.bucket)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete resume object from minio: %w", err)
	}

	return nil
}

func (s *minioResumeStorage) ensureBucket(region string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket %q: %w", s.bucket, err)
	}
	if exists {
		return nil
	}

	err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: region})
	if err == nil {
		return nil
	}

	response := minio.ToErrorResponse(err)
	if response.Code == "BucketAlreadyOwnedByYou" || response.Code == "BucketAlreadyExists" {
		return nil
	}

	return fmt.Errorf("create minio bucket %q: %w", s.bucket, err)
}

func fileExtension(fileName, contentType string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if ext != "" {
		return ext
	}

	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "application/pdf":
		return ".pdf"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}

func fallbackContentType(fileName string) string {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func parseMinIOPath(minIOPath, defaultBucket string) (string, string, error) {
	trimmed := strings.TrimSpace(minIOPath)
	if trimmed == "" {
		return "", "", fmt.Errorf("minio path is empty")
	}

	trimmed = strings.TrimPrefix(trimmed, "minio://")
	slashIndex := strings.Index(trimmed, "/")
	if slashIndex <= 0 || slashIndex >= len(trimmed)-1 {
		return "", "", fmt.Errorf("invalid minio path")
	}

	bucket := strings.TrimSpace(trimmed[:slashIndex])
	objectKey := strings.TrimSpace(trimmed[slashIndex+1:])
	if bucket == "" {
		bucket = strings.TrimSpace(defaultBucket)
	}
	if bucket == "" || objectKey == "" {
		return "", "", fmt.Errorf("invalid minio path")
	}

	return bucket, objectKey, nil
}

func resolveOriginalFileName(userMetadata map[string]string, objectKey string) string {
	for key, value := range userMetadata {
		if !strings.Contains(strings.ToLower(key), "original-filename") {
			continue
		}
		candidate := strings.TrimSpace(value)
		if candidate != "" {
			return filepath.Base(candidate)
		}
	}

	fileName := filepath.Base(strings.TrimSpace(objectKey))
	if fileName == "." || fileName == "/" || fileName == "" {
		return "resume-download"
	}

	return fileName
}

func normalizePathSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	builder := strings.Builder{}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteRune('-')
		}
	}

	normalized := strings.Trim(builder.String(), "-")
	normalized = strings.ReplaceAll(normalized, "--", "-")
	return normalized
}
