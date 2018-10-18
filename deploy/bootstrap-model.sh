#!/bin/bash

# Bootstrap a model

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source ${SCRIPT_DIR}/common.sh
source ${SCRIPT_DIR}/utils.sh

echo "Bootstrapping an agogo model!"
echo "Bucket name:      ${BUCKET_NAME}"
echo "Board Size:       ${BOARD_SIZE}"

MODEL_NAME=000000-bootstrap
GOPATH=$SCRIPT_DIR/..

{agogo} bootstrap s3://$BUCKET_NAME/models/$MODEL_NAME
