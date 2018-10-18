#!/bin/bash

set -e

echo bucket: s3://$BUCKET_NAME
echo board_size: $BOARD_SIZE
echo ""

pwd
dd if=/dev/zero of=test.out  bs=1M  count=1
aws s3 cp test.out s3://$BUCKET_NAME/
echo "Copied test to S3!"
