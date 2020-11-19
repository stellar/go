# sep10-homedomaincheck

This tool checks a space separated list of home domains for if they support SEP-10 v3's specification of what a home domain is. It verifies that a challenge transaction issued by the home domains web auth endpoint contains the home domain.

Outputs a CSV to stdout containing for each home domain if it passes or fails, and if it fails what the error is, and regardless, what the challenge transaction was the tool received.

## Install

```
go get github.com/stellar/go/tools/sep10-homedomaincheck
```

## Usage

```
sep10-homedomaincheck [homedomain] [homedomain] ...
```
