#!/bin/bash -e

# Move to repo root
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/../../.."
# module name is the sub-folder name under ./build
MODULE=$1
DOCKER_REPO_PREFIX=$2
DOCKER_PUSH=$3

if [ ! -z "$DOCKER_REPO_PREFIX" ]; then
   DOCKER_REPO_PREFIX="$DOCKER_REPO_PREFIX/" 
fi

build_target () { 
    DOCKER_TAG="$DOCKER_REPO_PREFIX"lighthorizon-"$MODULE_NAME"
    docker build --tag $DOCKER_TAG --platform linux/amd64 -f "exp/lighthorizon/build/$MODULE_NAME/Dockerfile" . 
    if [ "$DOCKER_PUSH" == "true" ]; then
        docker push $DOCKER_TAG
    fi
}

MODULE_NAME=$MODULE
case $MODULE in
index_batch)
    build_target
    ;;
ledgerexporter)
    build_target
    ;;
index_single)
    build_target
    ;;
web)
    build_target
    ;;
all)
    MODULE_NAME=index-batch
    build_target
    MODULE_NAME=web
    build_target
    MODULE_NAME=index-single
    build_target
    MODULE_NAME=ledgerexporter
    build_target
    ;;  
*)
    echo -n "unknown MODULE build parameter, must be one of all|index_batch|web|index_single|ledgerexporter"
    exit 1
    ;;
esac

