package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2authorizers"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awscloudformation "github.com/aws/aws-sdk-go-v2/service/cloudformation"
	awslambdasdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type InfraStackProps struct {
	awscdk.StackProps
	ImageURI string
}

const (
	stackName                   = "InfraStack"
	linkPreviewAPIKeySecretName = "debugjois-dev/linkpreview-api-key"
	deepgramAPIKeySecretName    = "debugjois-dev/deepgram-api-key"
	siteBucketARN               = "arn:aws:s3:::debugjois-dev-site"
)

func NewInfraStack(scope constructs.Construct, id string, props *InfraStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	imageURI := ""
	if props != nil {
		sprops = props.StackProps
		imageURI = props.ImageURI
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Create dedicated ECR repo with lifecycle policy
	repo := awsecr.NewRepository(stack, jsii.String("DebugJoisDevRepo"), &awsecr.RepositoryProps{
		RepositoryName: jsii.String("debugjois-dev"),
		RemovalPolicy:  awscdk.RemovalPolicy_DESTROY,
		EmptyOnDelete:  jsii.Bool(true),
	})
	repo.AddLifecycleRule(&awsecr.LifecycleRule{
		MaxImageCount: jsii.Number(3),
		Description:   jsii.String("Keep only last 3 images"),
	})

	linkPreviewAPIKeySecret := awssecretsmanager.NewCfnSecret(stack, jsii.String("LinkPreviewAPIKeySecret"), &awssecretsmanager.CfnSecretProps{
		Name:        jsii.String(linkPreviewAPIKeySecretName),
		Description: jsii.String("LinkPreview API key for the debugjois.dev backend"),
	})
	linkPreviewAPIKeySecret.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY, nil)

	deepgramAPIKeySecret := awssecretsmanager.NewCfnSecret(stack, jsii.String("DeepgramAPIKeySecret"), &awssecretsmanager.CfnSecretProps{
		Name:        jsii.String(deepgramAPIKeySecretName),
		Description: jsii.String("Deepgram API key for the debugjois.dev backend"),
	})
	deepgramAPIKeySecret.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY, nil)

	linkPreviewAPIKeySecretRef := awssecretsmanager.Secret_FromSecretCompleteArn(
		stack,
		jsii.String("LinkPreviewAPIKeySecretRef"),
		linkPreviewAPIKeySecret.AttrId(),
	)

	deepgramAPIKeySecretRef := awssecretsmanager.Secret_FromSecretCompleteArn(
		stack,
		jsii.String("DeepgramAPIKeySecretRef"),
		deepgramAPIKeySecret.AttrId(),
	)

	imageRepoName, imageTagOrDigest, err := resolveImageReference(context.Background(), imageURI)
	if err != nil {
		panic(err)
	}

	imageRepo := awsecr.Repository_FromRepositoryName(stack, jsii.String("LambdaImageRepo"), jsii.String(imageRepoName))
	siteBucket := awss3.Bucket_FromBucketArn(stack, jsii.String("DebugJoisDevSiteBucket"), jsii.String(siteBucketARN))

	githubOidcProvider := awsiam.NewCfnOIDCProvider(stack, jsii.String("GitHubOidcProvider"), &awsiam.CfnOIDCProviderProps{
		Url:          jsii.String("https://token.actions.githubusercontent.com"),
		ClientIdList: jsii.Strings("sts.amazonaws.com"),
	})
	githubOidcProvider.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)

	githubActionsRole := awsiam.NewRole(stack, jsii.String("GitHubActionsDeployRole"), &awsiam.RoleProps{
		RoleName: jsii.String("debugjois-dev-github-actions-role"),
		Description: jsii.String(
			"Role assumed by GitHub Actions via OIDC for debugjois.dev backend deployments",
		),
		AssumedBy: awsiam.NewOpenIdConnectPrincipal(githubOidcProvider, &map[string]interface{}{
			"StringEquals": map[string]interface{}{
				"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
				"token.actions.githubusercontent.com:sub": "repo:deepakjois/debugjois.dev:ref:refs/heads/main",
			},
		}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AdministratorAccess")),
		},
		MaxSessionDuration: awscdk.Duration_Hours(jsii.Number(1)),
	})

	// Create Lambda function from the Docker image asset
	fn := awslambda.NewDockerImageFunction(stack, jsii.String("DebugJoisDevLambda"), &awslambda.DockerImageFunctionProps{
		Code: awslambda.DockerImageCode_FromEcr(imageRepo, &awslambda.EcrImageCodeProps{
			TagOrDigest: jsii.String(imageTagOrDigest),
		}),
		Architecture: awslambda.Architecture_X86_64(),
		MemorySize:   jsii.Number(512),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Description:  jsii.String("debugjois.dev Lambda function"),
		Environment: &map[string]*string{
			"LINKPREVIEW_API_KEY_SECRET_ARN": linkPreviewAPIKeySecret.AttrId(),
			"DEEPGRAM_API_KEY_SECRET_ARN":    deepgramAPIKeySecret.AttrId(),
			"GOOGLE_APPLICATION_CREDENTIALS": jsii.String("/gcp-credentials.json"),
		},
	})
	linkPreviewAPIKeySecretRef.GrantRead(fn, nil)
	deepgramAPIKeySecretRef.GrantRead(fn, nil)
	siteBucket.GrantReadWrite(fn, nil)
	fn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("lambda:InvokeFunction"),
		Resources: jsii.Strings("*"),
	}))

	// Create HTTP API Gateway with Lambda proxy integration
	lambdaIntegration := awsapigatewayv2integrations.NewHttpLambdaIntegration(
		jsii.String("LambdaIntegration"), fn,
		&awsapigatewayv2integrations.HttpLambdaIntegrationProps{},
	)

	jwtAuthorizer := awsapigatewayv2authorizers.NewHttpJwtAuthorizer(
		jsii.String("GoogleJwtAuthorizer"),
		jsii.String("https://accounts.google.com"),
		&awsapigatewayv2authorizers.HttpJwtAuthorizerProps{
			JwtAudience: jsii.Strings(
				"1056519509576-d3942d1vuamh6as450jf0ds1qct288c5.apps.googleusercontent.com",
			),
		},
	)

	httpApi := awsapigatewayv2.NewHttpApi(stack, jsii.String("DebugJoisDevApi"), &awsapigatewayv2.HttpApiProps{
		ApiName: jsii.String("debugjois-dev-api"),
		CorsPreflight: &awsapigatewayv2.CorsPreflightOptions{
			AllowOrigins: jsii.Strings("*"),
			AllowHeaders: jsii.Strings("content-type", "x-amz-date", "authorization", "x-api-key", "x-amz-security-token"),
			AllowMethods: &[]awsapigatewayv2.CorsHttpMethod{
				awsapigatewayv2.CorsHttpMethod_GET,
				awsapigatewayv2.CorsHttpMethod_POST,
				awsapigatewayv2.CorsHttpMethod_PUT,
				awsapigatewayv2.CorsHttpMethod_PATCH,
				awsapigatewayv2.CorsHttpMethod_DELETE,
				awsapigatewayv2.CorsHttpMethod_HEAD,
				awsapigatewayv2.CorsHttpMethod_OPTIONS,
			},
			MaxAge: awscdk.Duration_Days(jsii.Number(10)),
		},
		Description: jsii.String("debugjois.dev HTTP API"),
	})

	// Add method-specific routes with the JWT authorizer.
	// Using explicit methods (not $default catch-all) so OPTIONS preflight
	// is handled by the built-in CORS handler without hitting the authorizer.
	authMethods := &[]awsapigatewayv2.HttpMethod{
		awsapigatewayv2.HttpMethod_GET,
		awsapigatewayv2.HttpMethod_POST,
		awsapigatewayv2.HttpMethod_PUT,
		awsapigatewayv2.HttpMethod_PATCH,
		awsapigatewayv2.HttpMethod_DELETE,
		awsapigatewayv2.HttpMethod_HEAD,
	}
	routeOpts := &awsapigatewayv2.AddRoutesOptions{
		Path:        jsii.String("/{proxy+}"),
		Methods:     authMethods,
		Integration: lambdaIntegration,
		Authorizer:  jwtAuthorizer,
	}
	httpApi.AddRoutes(routeOpts)
	httpApi.AddRoutes(&awsapigatewayv2.AddRoutesOptions{
		Path:        jsii.String("/"),
		Methods:     authMethods,
		Integration: lambdaIntegration,
		Authorizer:  jwtAuthorizer,
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("LambdaFunctionArn"), &awscdk.CfnOutputProps{
		Value:       fn.FunctionArn(),
		Description: jsii.String("Lambda function ARN"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("LambdaFunctionName"), &awscdk.CfnOutputProps{
		Value:       fn.FunctionName(),
		Description: jsii.String("Lambda function name"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("EcrRepositoryUri"), &awscdk.CfnOutputProps{
		Value:       repo.RepositoryUri(),
		Description: jsii.String("ECR repository URI"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("LinkPreviewAPIKeySecretArn"), &awscdk.CfnOutputProps{
		Value:       linkPreviewAPIKeySecret.AttrId(),
		Description: jsii.String("Secrets Manager ARN for the LinkPreview API key"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("LinkPreviewAPIKeySecretName"), &awscdk.CfnOutputProps{
		Value:       jsii.String(linkPreviewAPIKeySecretName),
		Description: jsii.String("Secrets Manager name for the LinkPreview API key"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("DeepgramAPIKeySecretArn"), &awscdk.CfnOutputProps{
		Value:       deepgramAPIKeySecret.AttrId(),
		Description: jsii.String("Secrets Manager ARN for the Deepgram API key"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("DeepgramAPIKeySecretName"), &awscdk.CfnOutputProps{
		Value:       jsii.String(deepgramAPIKeySecretName),
		Description: jsii.String("Secrets Manager name for the Deepgram API key"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("GitHubActionsRoleArn"), &awscdk.CfnOutputProps{
		Value:       githubActionsRole.RoleArn(),
		Description: jsii.String("IAM role ARN for GitHub Actions OIDC deployments"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("ApiUrl"), &awscdk.CfnOutputProps{
		Value:       httpApi.Url(),
		Description: jsii.String("HTTP API Gateway URL"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("SiteBucketArn"), &awscdk.CfnOutputProps{
		Value:       jsii.String(siteBucketARN),
		Description: jsii.String("S3 bucket ARN for debugjois.dev static assets and transcripts"),
	})

	return stack
}

func main() {
	defer jsii.Close()

	imageURI := flag.String("image-uri", "", "Explicit ECR image URI to deploy")
	flag.Parse()

	app := awscdk.NewApp(nil)

	NewInfraStack(app, stackName, &InfraStackProps{
		StackProps: awscdk.StackProps{
			Env:         env(),
			Synthesizer: NewNoAssumeRoleSynthesizer(nil),
		},
		ImageURI: *imageURI,
	})

	app.Synth(nil)
}

// env returns the AWS environment using CDK default account and region from CLI config.
func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}

func resolveImageReference(ctx context.Context, explicitImageURI string) (string, string, error) {
	imageURI := strings.TrimSpace(explicitImageURI)
	if imageURI == "" {
		var err error
		imageURI, err = lookupDeployedImageURI(ctx)
		if err != nil {
			return "", "", fmt.Errorf("resolve lambda image: %w", err)
		}
	}

	imageRepoName, imageTagOrDigest, err := parseEcrImageURI(imageURI)
	if err != nil {
		return "", "", err
	}

	return imageRepoName, imageTagOrDigest, nil
}

func lookupDeployedImageURI(ctx context.Context) (string, error) {
	region := strings.TrimSpace(os.Getenv("CDK_DEFAULT_REGION"))
	loadOptions := []func(*config.LoadOptions) error{}
	if region != "" {
		loadOptions = append(loadOptions, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return "", fmt.Errorf("load AWS config: %w", err)
	}

	cloudFormationClient := awscloudformation.NewFromConfig(cfg)
	stackOutput, err := cloudFormationClient.DescribeStacks(ctx, &awscloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return "", fmt.Errorf("describe stack %s: %w", stackName, err)
	}
	if len(stackOutput.Stacks) == 0 {
		return "", fmt.Errorf("stack %s not found", stackName)
	}

	functionName := ""
	for _, output := range stackOutput.Stacks[0].Outputs {
		if aws.ToString(output.OutputKey) == "LambdaFunctionName" {
			functionName = aws.ToString(output.OutputValue)
			break
		}
	}
	if functionName == "" {
		return "", fmt.Errorf("stack %s does not expose LambdaFunctionName output", stackName)
	}

	lambdaClient := awslambdasdk.NewFromConfig(cfg)
	functionOutput, err := lambdaClient.GetFunction(ctx, &awslambdasdk.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return "", fmt.Errorf("get Lambda function %s: %w", functionName, err)
	}

	imageURI := strings.TrimSpace(aws.ToString(functionOutput.Code.ImageUri))
	if imageURI == "" {
		return "", fmt.Errorf("lambda function %s does not have an image URI", functionName)
	}

	return imageURI, nil
}

func parseEcrImageURI(imageURI string) (string, string, error) {
	parts := strings.SplitN(imageURI, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid image URI %q: expected ECR URI", imageURI)
	}

	repoAndRef := parts[1]
	if repoParts := strings.SplitN(repoAndRef, "@", 2); len(repoParts) == 2 {
		if repoParts[0] == "" || repoParts[1] == "" {
			return "", "", fmt.Errorf("invalid image URI %q: expected repository and digest", imageURI)
		}
		return repoParts[0], repoParts[1], nil
	}

	lastColon := strings.LastIndex(repoAndRef, ":")
	if lastColon == -1 {
		return "", "", fmt.Errorf("invalid image URI %q: expected tag or digest", imageURI)
	}

	repoName := repoAndRef[:lastColon]
	tag := repoAndRef[lastColon+1:]
	if repoName == "" || tag == "" {
		return "", "", fmt.Errorf("invalid image URI %q: expected repository and tag", imageURI)
	}

	return repoName, tag, nil
}
