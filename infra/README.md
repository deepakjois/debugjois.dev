# Infrastructure

AWS CDK app for the backend infrastructure.

## Requirements

- Go 1.26+
- Node.js and the AWS CDK CLI
- AWS credentials in the default profile
- an explicit image URI argument when deploying a specific image

## Common commands

Run from `infra/`:

```bash
cdk diff
cdk --app 'go mod download && go run infra.go --image-uri <ecr-image-uri-or-digest>' deploy --require-approval never
cdk synth
```

If no explicit image URI argument is provided, `cdk ls`, `cdk synth`, `cdk diff`,
and plain `cdk deploy` fall back to the image currently deployed on the Lambda.

## Deploy helper

From the repository root:

```bash
./infra/deploy.sh
./infra/deploy.sh --build-image
```

- `./infra/deploy.sh` lets `infra.go` reuse the image currently configured on the deployed Lambda
- `./infra/deploy.sh --build-image` first runs `./backend/build-and-push-image.sh` and passes the resulting image URI directly to `infra.go`

## Notes

- `infra.go` falls back to the currently deployed Lambda image when no `--image-uri` argument is provided
- the stack outputs include `ApiUrl`, `LambdaFunctionName`, and `EcrRepositoryUri`
- `cloudfront/domain-redirect-debugjois-dev.js` is the checked-in source for the live CloudFront Function that redirects `debugjois.dev` to `www.debugjois.dev` and rewrites `/app` SPA routes to `/app/index.html`
- deploy CloudFront Function updates with `aws cloudfront update-function --name domain-redirect-debugjois-dev --if-match <etag> --function-config Comment="Redirect debugjois.dev -> www.debugjois.dev and rewrite /app SPA paths",Runtime=cloudfront-js-2.0 --function-code fileb://"$(pwd)/cloudfront/domain-redirect-debugjois-dev.js"` and then `aws cloudfront publish-function --name domain-redirect-debugjois-dev --if-match <etag>` from `infra/`
