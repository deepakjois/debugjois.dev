# CloudFront Function

`domain-redirect-debugjois-dev.js` is the source for the live CloudFront Function named `domain-redirect-debugjois-dev`.

The function redirects `debugjois.dev` to `www.debugjois.dev` and leaves all other requests unchanged.

## Deploy

From the repository root:

```bash
etag=$(aws cloudfront describe-function \
  --name domain-redirect-debugjois-dev \
  --stage DEVELOPMENT \
  --query ETag \
  --output text)

aws cloudfront update-function \
  --name domain-redirect-debugjois-dev \
  --if-match "$etag" \
  --function-config Comment="Redirect debugjois.dev -> www.debugjois.dev",Runtime=cloudfront-js-2.0 \
  --function-code fileb://site/cloudfront/domain-redirect-debugjois-dev.js

etag=$(aws cloudfront describe-function \
  --name domain-redirect-debugjois-dev \
  --stage DEVELOPMENT \
  --query ETag \
  --output text)

aws cloudfront publish-function \
  --name domain-redirect-debugjois-dev \
  --if-match "$etag"
```
