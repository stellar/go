#!/usr/bin/env bash

go list -deps -test -f '{{with .Module}}{{.Path}} {{.Version}}{{end}}' ./... | sort -u
