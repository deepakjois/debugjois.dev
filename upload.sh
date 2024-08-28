#!/bin/bash

# S3 bucket name
S3_BUCKET="debugjois-dev-site"

# local folder path
LOCAL_FOLDER="build"

# Loop through all files in the local folder
for file in "$LOCAL_FOLDER"/*; do
    # Get the filename
    filename=$(basename "$file")

    # Get the file extension
    extension="${filename##*.}"

    # Set the content type
    if [ "$extension" = "$filename" ]; then
        # No extension, set MIME type to text/html
        content_type="text/html"
    else
        # Use the default content type based on the file extension
        content_type=""
    fi

    # Upload the file to S3
    if [ -n "$content_type" ]; then
        aws s3 cp "$file" "s3://$S3_BUCKET/$filename" --content-type "$content_type"
    else
        aws s3 cp "$file" "s3://$S3_BUCKET/$filename"
    fi
done

# Sync the images folder
aws s3 sync --dryrun "build/images" "s3://$S3_BUCKET/images"

echo "Upload complete!"
