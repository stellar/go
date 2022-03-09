#!/bin/bash -e

# Move to repo root
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/../.."

docker build --tag lighthorizon --platform linux/amd64 -f exp/lighthorizon/Dockerfile .
