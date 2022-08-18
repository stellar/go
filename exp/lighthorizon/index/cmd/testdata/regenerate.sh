#!/bin/bash

docker run -e WRITE_LATEST_PATH=true -e START=1410048 -e END=1410367 -e ARCHIVE_TARGET=file:///testdata/ -v $PWD:/tetsdata/ stellar/horizon-ledgerexporter:latest
