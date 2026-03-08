# Infrastructure

AWS CDK app for the backend infrastructure.

## Requirements

- Go 1.26+
- Node.js and the AWS CDK CLI
- AWS credentials in the default profile
- `IMAGE_URI` for deploys that should use a specific image

## Common commands

Run from `infra/`:

```bash
cdk diff
IMAGE_URI=<ecr-image-uri-or-digest> cdk deploy --require-approval never
cdk synth
```

If `IMAGE_URI` is unset, `cdk ls`, `cdk synth`, and `cdk diff` fall back to the
image currently deployed on the Lambda.

## Deploy helper

From the repository root:

```bash
./infra/deploy.sh
./infra/deploy.sh --build-image
```

- `./infra/deploy.sh` lets `infra.go` reuse the image currently configured on the deployed Lambda
- `./infra/deploy.sh --build-image` first runs `./backend/build-and-push-image.sh`

## Notes

- `infra.go` falls back to the currently deployed Lambda image when `IMAGE_URI` is unset
- the stack outputs include `ApiUrl`, `LambdaFunctionName`, and `EcrRepositoryUri`
