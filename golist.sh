#!/usr/bin/env bash

go list -deps -test -f '{{with .Module}}{{.Path}} {{.Version}}{{end}}' ./... | LC_ALL=C sort -u
