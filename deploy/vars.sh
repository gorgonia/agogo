#!/bin/bash

# Game vars
export BOARD_SIZE="19"

# AWS vars
export BUCKET_NAME="agogo-models"
export MASTERTYPE="m3.medium"
export SLAVETYPE="t2.medium"
export SLAVES="2"
export ZONE="ap-southeast-2b"

# K8s vars
export NAME="agogo.k8s.local"
export KOPS_STATE_STORE="s3://agogo-cluster"
export PROJECT="agogo"
export CLUSTER_NAME=$PROJECT


# Docker vars
export VERSION_TAG="0.1"
export GPU_PLAYER_CONTAINER="agogo-gpu-player"
export CPU_PLAYER_CONTAINER="agogo-player"
