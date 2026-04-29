# debugjois.dev

Static site generator and content for [debugjois.dev](https://debugjois.dev).

## Layout

- `site/` - Go static site generator for the main website and daily log
- `site/content/` - source content, including daily notes
- `site/templates/` - HTML templates
- `site/static/` - static assets copied into the build output
- `site/cloudfront/` - source and deploy notes for the live CloudFront Function

## Workspace

The Go workspace in `go.work` includes only `./site`.

## Common commands

Run from `site/` unless noted otherwise:

- `go build -o debugjois-site .` - build the site binary
- `./debugjois-site build` - build the static site into `build/`
- `./debugjois-site build --dev` - include drafts and scratch content
- `./debugjois-site build --rebuild` - rebuild the entire archive
- `./debugjois-site sync-notes-obsidian` - sync daily notes from Google Drive shared drive
- `./debugjois-site commit-notes` - commit daily note changes
- `./debugjois-site commit-notes --skip-ci` - commit with `[skip ci]` appended to the message
- `./debugjois-site upload` - upload generated files to S3
- `./debugjois-site upload --dryrun` - preview upload without writing to S3
- `./debugjois-site build-newsletter` - preview the weekly newsletter
- `./debugjois-site build-newsletter --post` - post newsletter draft to Buttondown
- `./debugjois-site build-newsletter --post --notify` - post and notify via Resend
- `go test ./...` - run site tests
- `golangci-lint run` - run the configured Go lint/format checks

## Deployments

- Site deploys are handled by the site GitHub workflows and `./debugjois-site upload`.
- CloudFront Function source is tracked in `site/cloudfront/`; see `site/cloudfront/README.md` for manual deploy commands.

## AWS GitHub Actions OIDC setup

The site workflows use `aws-actions/configure-aws-credentials` with the repo secret
`AWS_ROLE_ARN` to upload generated files to `s3://debugjois-dev-site`. Keep this
OIDC provider and role outside any application stack so deleting app/backend
infrastructure does not break site deploys.

Use these commands to recreate the provider and site deploy role if they are
missing:

```bash
ACCOUNT_ID=654654546088
REPO=deepakjois/debugjois.dev
ROLE_NAME=debugjois-dev-site-github-actions-role
BUCKET=debugjois-dev-site

# IAM can retrieve the GitHub Actions OIDC thumbprint automatically. If the
# provider already exists, this command fails harmlessly with EntityAlreadyExists.
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com

cat > /tmp/debugjois-dev-site-github-actions-trust.json <<EOF_TRUST
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:${REPO}:ref:refs/heads/main"
        }
      }
    }
  ]
}
EOF_TRUST

aws iam create-role \
  --role-name "$ROLE_NAME" \
  --assume-role-policy-document file:///tmp/debugjois-dev-site-github-actions-trust.json

cat > /tmp/debugjois-dev-site-github-actions-s3-policy.json <<EOF_POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::${BUCKET}"
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::${BUCKET}/*"
    }
  ]
}
EOF_POLICY

aws iam put-role-policy \
  --role-name "$ROLE_NAME" \
  --policy-name debugjois-dev-site-s3-deploy \
  --policy-document file:///tmp/debugjois-dev-site-github-actions-s3-policy.json

ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"
gh secret set AWS_ROLE_ARN --repo "$REPO" --body "$ROLE_ARN"
```

Verify the setup:

```bash
aws iam list-open-id-connect-providers
aws iam get-role --role-name debugjois-dev-site-github-actions-role \
  --query 'Role.AssumeRolePolicyDocument'
gh secret list --repo deepakjois/debugjois.dev | grep AWS_ROLE_ARN
```
