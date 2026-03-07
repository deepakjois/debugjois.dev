package main

import (
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
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type InfraStackProps struct {
	awscdk.StackProps
}

func NewInfraStack(scope constructs.Construct, id string, props *InfraStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
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

	imageURI := os.Getenv("IMAGE_URI")
	if imageURI == "" {
		panic("IMAGE_URI must be set to an existing ECR image URI")
	}

	imageRepoName, imageTagOrDigest, err := parseEcrImageURI(imageURI)
	if err != nil {
		panic(err)
	}

	imageRepo := awsecr.Repository_FromRepositoryName(stack, jsii.String("LambdaImageRepo"), jsii.String(imageRepoName))

	githubOidcProvider := awsiam.NewOpenIdConnectProvider(stack, jsii.String("GitHubOidcProvider"), &awsiam.OpenIdConnectProviderProps{
		Url:       jsii.String("https://token.actions.githubusercontent.com"),
		ClientIds: jsii.Strings("sts.amazonaws.com"),
	})

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
	})

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
			AllowHeaders: jsii.Strings("Content-Type", "X-Amz-Date", "Authorization", "X-Api-Key", "X-Amz-Security-Token"),
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
	awscdk.NewCfnOutput(stack, jsii.String("GitHubActionsRoleArn"), &awscdk.CfnOutputProps{
		Value:       githubActionsRole.RoleArn(),
		Description: jsii.String("IAM role ARN for GitHub Actions OIDC deployments"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("ApiUrl"), &awscdk.CfnOutputProps{
		Value:       httpApi.Url(),
		Description: jsii.String("HTTP API Gateway URL"),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewInfraStack(app, "InfraStack", &InfraStackProps{
		awscdk.StackProps{
			Env: env(),
		},
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

func parseEcrImageURI(imageURI string) (string, string, error) {
	parts := strings.SplitN(imageURI, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid IMAGE_URI %q: expected ECR URI", imageURI)
	}

	repoAndRef := parts[1]
	if repoParts := strings.SplitN(repoAndRef, "@", 2); len(repoParts) == 2 {
		if repoParts[0] == "" || repoParts[1] == "" {
			return "", "", fmt.Errorf("invalid IMAGE_URI %q: expected repository and digest", imageURI)
		}
		return repoParts[0], repoParts[1], nil
	}

	lastColon := strings.LastIndex(repoAndRef, ":")
	if lastColon == -1 {
		return "", "", fmt.Errorf("invalid IMAGE_URI %q: expected tag or digest", imageURI)
	}

	repoName := repoAndRef[:lastColon]
	tag := repoAndRef[lastColon+1:]
	if repoName == "" || tag == "" {
		return "", "", fmt.Errorf("invalid IMAGE_URI %q: expected repository and tag", imageURI)
	}

	return repoName, tag, nil
}
