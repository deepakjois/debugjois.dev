package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	secretsmanagersdk "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/joho/godotenv"
)

const (
	githubTokenEnvVar                = "GITHUB_TOKEN"
	githubPATSecretARNEnvVar         = "GITHUB_PAT_SECRET_ARN"
	linkPreviewAPIKeyEnvVar          = "LINKPREVIEW_API_KEY"
	linkPreviewAPIKeySecretARNEnvVar = "LINKPREVIEW_API_KEY_SECRET_ARN"
	linkPreviewBaseURL               = "https://api.linkpreview.net"
	defaultLocalDotEnvPath           = ".env"
)

func loadLocalEnvFile() error {
	if err := godotenv.Overload(defaultLocalDotEnvPath); err != nil {
		return fmt.Errorf("load local env file %q: %w", defaultLocalDotEnvPath, err)
	}

	if strings.TrimSpace(os.Getenv(githubTokenEnvVar)) == "" {
		return fmt.Errorf("%s must be set in %s for local development", githubTokenEnvVar, defaultLocalDotEnvPath)
	}

	if strings.TrimSpace(os.Getenv(linkPreviewAPIKeyEnvVar)) == "" {
		return fmt.Errorf("%s must be set in %s for local development", linkPreviewAPIKeyEnvVar, defaultLocalDotEnvPath)
	}

	return nil
}

func fetchSecretValue(ctx context.Context, arn string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load AWS config for secret %q: %w", arn, err)
	}

	client := secretsmanagersdk.NewFromConfig(cfg)
	output, err := client.GetSecretValue(ctx, &secretsmanagersdk.GetSecretValueInput{
		SecretId: &arn,
	})
	if err != nil {
		return "", fmt.Errorf("get secret value for %q: %w", arn, err)
	}

	value := strings.TrimSpace(aws.ToString(output.SecretString))
	if value == "" {
		return "", fmt.Errorf("secret %q is empty", arn)
	}

	return value, nil
}

func loadLambdaGitHubToken(ctx context.Context) error {
	secretARN := strings.TrimSpace(os.Getenv(githubPATSecretARNEnvVar))
	if secretARN == "" {
		return fmt.Errorf("%s must be set in Lambda", githubPATSecretARNEnvVar)
	}

	token, err := fetchSecretValue(ctx, secretARN)
	if err != nil {
		return fmt.Errorf("load GitHub token: %w", err)
	}

	if err := os.Setenv(githubTokenEnvVar, token); err != nil {
		return fmt.Errorf("set %s: %w", githubTokenEnvVar, err)
	}

	return nil
}

func loadLambdaLinkPreviewAPIKey(ctx context.Context) error {
	secretARN := strings.TrimSpace(os.Getenv(linkPreviewAPIKeySecretARNEnvVar))
	if secretARN == "" {
		return fmt.Errorf("%s must be set in Lambda", linkPreviewAPIKeySecretARNEnvVar)
	}

	apiKey, err := fetchSecretValue(ctx, secretARN)
	if err != nil {
		return fmt.Errorf("load LinkPreview API key: %w", err)
	}

	if err := os.Setenv(linkPreviewAPIKeyEnvVar, apiKey); err != nil {
		return fmt.Errorf("set %s: %w", linkPreviewAPIKeyEnvVar, err)
	}

	return nil
}

func loadLambdaSecrets(ctx context.Context) error {
	if err := loadLambdaGitHubToken(ctx); err != nil {
		return err
	}

	if err := loadLambdaLinkPreviewAPIKey(ctx); err != nil {
		return err
	}

	return nil
}
