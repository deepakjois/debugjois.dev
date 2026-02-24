package main

import (
	"os"
	"path"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2authorizers"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecrassets"
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

	// Build and push Lambda image from the api/ directory
	dirName, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	lambdaDir := path.Join(dirName, "..", "api")

	imageAsset := awsecrassets.NewDockerImageAsset(stack, jsii.String("DebugJoisDevImage"), &awsecrassets.DockerImageAssetProps{
		Directory: jsii.String(lambdaDir),
		Platform:  awsecrassets.Platform_LINUX_AMD64(),
	})

	// Create Lambda function from the Docker image asset
	fn := awslambda.NewDockerImageFunction(stack, jsii.String("DebugJoisDevLambda"), &awslambda.DockerImageFunctionProps{
		Code: awslambda.DockerImageCode_FromEcr(imageAsset.Repository(), &awslambda.EcrImageCodeProps{
			TagOrDigest: imageAsset.ImageTag(),
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
