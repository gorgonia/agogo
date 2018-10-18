#!/bin/bash

## Kill the cluster with kops

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source ${SCRIPT_DIR}/vars.sh
source ${SCRIPT_DIR}/utils.sh

echo "Deleting cluster $NAME"
kops delete cluster $NAME --yes
