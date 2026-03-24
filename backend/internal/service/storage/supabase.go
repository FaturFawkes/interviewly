package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/logger"
	"go.uber.org/zap"
)

type supabaseResumeStorage struct {
	endpoint   string
	region     string
	accessKey  string
	secretKey  string
	bucket     string
	pathPrefix string
	s3Client   *s3.Client
}

func NewSupabaseResumeStorage(cfg *config.Config) (domain.ResumeFileStorage, error) {
	endpoint := normalizeSupabaseS3Endpoint(strings.TrimSpace(cfg.SupabaseS3Endpoint), strings.TrimSpace(cfg.SupabaseURL))
	region := strings.TrimSpace(cfg.SupabaseS3Region)
	if region == "" {
		region = "us-east-1"
	}

	accessKey := strings.TrimSpace(cfg.SupabaseS3AccessKeyID)
	secretKey := strings.TrimSpace(cfg.SupabaseS3SecretAccessKey)
	bucket := strings.TrimSpace(cfg.SupabaseStorageBucket)
	pathPrefix := strings.Trim(strings.TrimSpace(cfg.SupabaseStoragePathPrefix), "/")

	if endpoint == "" && accessKey == "" && secretKey == "" && bucket == "" {
		return nil, nil
	}

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("supabase s3 storage is not fully configured")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize supabase s3 config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = true
		options.BaseEndpoint = &endpoint
	})

	return &supabaseResumeStorage{
		endpoint:   endpoint,
		region:     region,
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucket:     bucket,
		pathPrefix: pathPrefix,
		s3Client:   s3Client,
	}, nil
}

func (s *supabaseResumeStorage) UploadResume(userID, fileName, contentType string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("resume file is empty")
	}

	logger.L().Info("[storage/supabase] uploading resume",
		zap.String("userID", userID),
		zap.String("fileName", fileName),
		zap.String("contentType", contentType),
		zap.Int("sizeBytes", len(data)),
	)

	userSegment := normalizePathSegment(userID)
	if userSegment == "" {
		userSegment = "anonymous"
	}

	extension := fileExtension(fileName, contentType)
	objectKey := fmt.Sprintf(
		"%sresumes/%s/%s-%s%s",
		s.objectKeyPrefix(),
		userSegment,
		time.Now().UTC().Format("20060102T150405"),
		uuid.NewString(),
		extension,
	)

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		logger.L().Error("[storage/supabase] upload failed", zap.String("userID", userID), zap.Error(err))
		return "", fmt.Errorf("upload resume to supabase s3: %w", err)
	}

	storagePath := fmt.Sprintf("supabase://%s/%s", s.bucket, objectKey)
	logger.L().Info("[storage/supabase] upload success", zap.String("userID", userID), zap.String("path", storagePath))
	return storagePath, nil
}

func (s *supabaseResumeStorage) DownloadResume(storagePath string) (*domain.ResumeFile, error) {
	logger.L().Info("[storage/supabase] downloading resume", zap.String("path", storagePath))

	bucket, objectKey, err := parseSupabasePath(storagePath, s.bucket)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &objectKey,
	})
	if err != nil {
		logger.L().Error("[storage/supabase] download failed", zap.String("path", storagePath), zap.Error(err))
		return nil, fmt.Errorf("download resume from supabase s3: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read resume from supabase s3: %w", err)
	}

	fileName := filepath.Base(strings.TrimSpace(objectKey))
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "resume-download"
	}

	contentType := strings.TrimSpace(safeString(resp.ContentType))
	if contentType == "" {
		contentType = fallbackContentType(fileName)
	}

	logger.L().Info("[storage/supabase] download success", zap.String("path", storagePath), zap.String("fileName", fileName), zap.Int("sizeBytes", len(data)))
	return &domain.ResumeFile{
		FileName:    fileName,
		ContentType: contentType,
		Data:        data,
	}, nil
}

func (s *supabaseResumeStorage) DeleteResume(storagePath string) error {
	logger.L().Info("[storage/supabase] deleting resume", zap.String("path", storagePath))

	bucket, objectKey, err := parseSupabasePath(storagePath, s.bucket)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &objectKey,
	})
	if err != nil {
		logger.L().Error("[storage/supabase] delete failed", zap.String("path", storagePath), zap.Error(err))
		return fmt.Errorf("delete resume from supabase s3: %w", err)
	}

	logger.L().Info("[storage/supabase] delete success", zap.String("path", storagePath))
	return nil
}

func normalizeSupabaseS3Endpoint(endpoint, supabaseURL string) string {
	if endpoint != "" {
		return strings.TrimSuffix(endpoint, "/")
	}

	supabaseURL = strings.TrimSpace(supabaseURL)
	if supabaseURL == "" {
		return ""
	}

	parsed, err := url.Parse(supabaseURL)
	if err != nil || parsed.Host == "" {
		return ""
	}

	host := strings.TrimSpace(parsed.Host)
	if strings.HasSuffix(host, ".supabase.co") && !strings.Contains(host, ".storage.supabase.co") {
		host = strings.TrimSuffix(host, ".supabase.co") + ".storage.supabase.co"
	}

	scheme := strings.TrimSpace(parsed.Scheme)
	if scheme == "" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/storage/v1/s3", scheme, host)
}

func (s *supabaseResumeStorage) objectKeyPrefix() string {
	if s.pathPrefix == "" {
		return ""
	}
	return s.pathPrefix + "/"
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseSupabasePath(storagePath, defaultBucket string) (string, string, error) {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		return "", "", fmt.Errorf("storage path is empty")
	}

	trimmed = strings.TrimPrefix(trimmed, "supabase://")
	trimmed = strings.Trim(trimmed, "/")
	slashIndex := strings.Index(trimmed, "/")
	if slashIndex <= 0 || slashIndex >= len(trimmed)-1 {
		return "", "", fmt.Errorf("invalid storage path")
	}

	bucket := strings.TrimSpace(trimmed[:slashIndex])
	objectKey := path.Clean("/" + strings.TrimSpace(trimmed[slashIndex+1:]))
	objectKey = strings.TrimPrefix(objectKey, "/")

	if bucket == "" {
		bucket = strings.TrimSpace(defaultBucket)
	}
	if bucket == "" || objectKey == "" || objectKey == "." {
		return "", "", fmt.Errorf("invalid storage path")
	}

	return bucket, objectKey, nil
}
