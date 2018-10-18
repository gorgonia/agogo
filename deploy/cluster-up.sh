#!/bin/bash

## Bring up the cluster with kops

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source ${SCRIPT_DIR}/vars.sh
source ${SCRIPT_DIR}/utils.sh

echo "Bringing up Kubernetes cluster"
echo "Using Cluster Name: ${CLUSTER_NAME}"
echo "Number of Nodes:    ${SLAVES}"
echo "Using Zone:         ${ZONE}"
echo "Bucket name:        ${BUCKET_NAME}"

export PARALLELISM="$((4 * ${SLAVES}))"

# Includes ugly workaround because kops is unable to take stdin as input to create -f, unlike kubectl
cat k8s_cluster.yaml | envsubst > k8s_cluster-edit.yaml && kops create -f k8s_cluster-edit.yaml
cat k8s_master.yaml | envsubst > k8s_master-edit.yaml && kops create -f k8s_master-edit.yaml
cat k8s_nodes.yaml | envsubst > k8s_nodes-edit.yaml && kops create -f k8s_nodes-edit.yaml

kops create secret --name $NAME sshpublickey admin -i ~/.ssh/id_rsa.pub
kops update cluster $NAME --yes

echo ""
echo "Cluster $NAME created!"
echo ""

# Cleanup from workaround
rm k8s_cluster-edit.yaml
rm k8s_master-edit.yaml
rm k8s_nodes-edit.yaml
