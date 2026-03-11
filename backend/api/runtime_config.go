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
	githubTokenEnvVar        = "GITHUB_TOKEN"
	githubPATSecretARNEnvVar = "GITHUB_PAT_SECRET_ARN"
	defaultLocalDotEnvPath   = ".env"
)

func loadLocalEnvFile() error {
	if err := godotenv.Overload(defaultLocalDotEnvPath); err != nil {
		return fmt.Errorf("load local env file %q: %w", defaultLocalDotEnvPath, err)
	}

	if strings.TrimSpace(os.Getenv(githubTokenEnvVar)) == "" {
		return fmt.Errorf("%s must be set in %s for local development", githubTokenEnvVar, defaultLocalDotEnvPath)
	}

	return nil
}

func loadLambdaGitHubToken(ctx context.Context) error {
	secretARN := strings.TrimSpace(os.Getenv(githubPATSecretARNEnvVar))
	if secretARN == "" {
		return fmt.Errorf("%s must be set in Lambda", githubPATSecretARNEnvVar)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config for secret %q: %w", secretARN, err)
	}

	client := secretsmanagersdk.NewFromConfig(cfg)
	output, err := client.GetSecretValue(ctx, &secretsmanagersdk.GetSecretValueInput{
		SecretId: &secretARN,
	})
	if err != nil {
		return fmt.Errorf("get secret value for %q: %w", secretARN, err)
	}

	token := strings.TrimSpace(aws.ToString(output.SecretString))
	if token == "" {
		return fmt.Errorf("secret %q did not contain a GitHub token", secretARN)
	}

	if err := os.Setenv(githubTokenEnvVar, token); err != nil {
		return fmt.Errorf("set %s: %w", githubTokenEnvVar, err)
	}

	return nil
}
