#!/bin/bash

docker run -e WRITE_LATEST_PATH=true -e START=1410048 -e END=1410367 -e ARCHIVE_TARGET=file:///fixtures/ -v $PWD:/fixtures/ stellar/horizon-ledgerexporter:latest
