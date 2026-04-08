package podcastaddict

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	transcriptObjectPrefix = "transcripts/"
	maxTranscriptSlugLen   = 120
)

var (
	transcriptSlugPattern     = regexp.MustCompile(`[^a-z0-9]+`)
	newTranscriptS3ClientFunc = newTranscriptS3Client
)

type transcriptS3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type transcriptRequest struct {
	Action  string `json:"action"`
	Podcast Result `json:"podcast"`
}

func PersistTranscript(ctx context.Context, bucketARN, action string, podcast Result, body []byte) error {
	bucketName, err := transcriptBucketNameFromARN(bucketARN)
	if err != nil {
		return err
	}

	key, err := transcriptObjectKey(action, podcast)
	if err != nil {
		return err
	}

	client, err := newTranscriptS3ClientFunc(ctx, bucketName)
	if err != nil {
		return err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("write transcript to s3://%s/%s: %w", bucketName, key, err)
	}

	return nil
}

func newTranscriptS3Client(ctx context.Context, bucketName string) (transcriptS3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config for transcript upload: %w", err)
	}

	region, err := manager.GetBucketRegion(ctx, s3.NewFromConfig(cfg), bucketName)
	if err != nil {
		return nil, fmt.Errorf("resolve transcript bucket region for %q: %w", bucketName, err)
	}
	cfg.Region = region

	return s3.NewFromConfig(cfg), nil
}

func transcriptBucketNameFromARN(bucketARN string) (string, error) {
	parsedARN, err := arn.Parse(strings.TrimSpace(bucketARN))
	if err != nil {
		return "", fmt.Errorf("parse transcript bucket ARN: %w", err)
	}
	if parsedARN.Service != "s3" || parsedARN.Resource == "" {
		return "", fmt.Errorf("transcript bucket ARN must be an S3 bucket ARN")
	}
	if strings.Contains(parsedARN.Resource, "/") {
		return "", fmt.Errorf("transcript bucket ARN must reference an S3 bucket, not an object")
	}

	return parsedARN.Resource, nil
}

func transcriptObjectKey(action string, podcast Result) (string, error) {
	payload, err := json.Marshal(transcriptRequest{
		Action:  action,
		Podcast: podcast,
	})
	if err != nil {
		return "", fmt.Errorf("marshal transcript payload hash input: %w", err)
	}

	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%s%s--%s.json", transcriptObjectPrefix, transcriptReadableSlug(podcast), hex.EncodeToString(sum[:])), nil
}

func transcriptReadableSlug(podcast Result) string {
	parts := []string{
		transcriptSlugPart(podcast.Podcast.Title),
		transcriptSlugPart(podcast.Episode.Title),
	}

	var slugParts []string
	for _, part := range parts {
		if part != "" {
			slugParts = append(slugParts, part)
		}
	}

	if len(slugParts) == 0 {
		for _, candidate := range []string{
			podcast.Source.ShareTitle,
			podcast.Source.EpisodeURL,
			podcast.Source.Input,
		} {
			part := transcriptSlugPart(candidate)
			if part != "" {
				slugParts = append(slugParts, part)
				break
			}
		}
	}

	slug := strings.Join(slugParts, "--")
	if slug == "" {
		slug = "podcast-transcript"
	}
	if len(slug) > maxTranscriptSlugLen {
		slug = strings.Trim(slug[:maxTranscriptSlugLen], "-")
	}
	if slug == "" {
		return "podcast-transcript"
	}

	return slug
}

func transcriptSlugPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	value = transcriptSlugPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
