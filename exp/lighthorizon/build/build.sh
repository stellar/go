#!/bin/bash -e

# Move to repo root
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/../../.."
# module name is the sub-folder name under ./build
MODULE=$1
DOCKER_REPO_PREFIX=$2
DOCKER_TAG=$3
DOCKER_PUSH=$4

if [ -z "$MODULE" ] ||\
   [ -z "$DOCKER_REPO_PREFIX" ] ||\
   [ -z "$DOCKER_TAG" ] ||\
   [ -z "$DOCKER_PUSH" ]; then
   echo "invalid parameters, requires './build.sh <service_name> <dockerhub_repo_name> <tag_name> <push_to_repo[true|false]>'"
   exit 1
fi

build_target () { 
    DOCKER_LABEL="$DOCKER_REPO_PREFIX"/lighthorizon-"$MODULE":"$DOCKER_TAG"
    docker build --tag $DOCKER_LABEL --platform linux/amd64 -f "exp/lighthorizon/build/$MODULE/Dockerfile" . 
    if [ "$DOCKER_PUSH" == "true" ]; then
        docker push $DOCKER_LABEL
    fi
}

case $MODULE in
index-batch)
    build_target
    ;;
ledgerexporter)
    build_target
    ;;
index-single)
    build_target
    ;;
web)
    build_target
    ;;
all)
    MODULE=index-batch
    build_target
    MODULE=web
    build_target
    MODULE=index-single
    build_target
    MODULE=ledgerexporter
    build_target
    ;;  
*)
    echo "unknown MODULE build parameter ('$MODULE'), must be one of all|index-batch|web|index-single|ledgerexporter"
    exit 1
    ;;
esac

