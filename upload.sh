#!/bin/bash
aws s3 sync --size-only build/ s3://debugjois-dev-site/ --exclude "index.html" --exclude "daily*" 
aws s3 sync --size-only build/ s3://debugjois-dev-site/ --exclude "*" --include "daily-archive-*" --content-type "text/html"
aws s3 sync build/ s3://debugjois-dev-site/ --exclude "*" --include "index.html" --include "daily" --include "daily.xml" --content-type "text/html"


echo "Upload complete!"
