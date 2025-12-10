# Migration Guide

## Migrating from github.com/stellar/go to github.com/stellar/go-stellar-sdk

The Golang SDKs are now located in a new dedicated repo for that sole purpose - https://github.com/stellar/go-stellar-sdk. 

Go client applications that were using SDK packages from `github.com/stellar/go` module will need to change their project to reference the new go module name `github.com/stellar/go-stellar-sdk` and rename package import statements from `github.com/stellar/go/..` to `github.com/stellar/go-stellar-sdk/..`


### 1. Update your go.mod
```bash
# Add the new module name
go get github.com/stellar/go-stellar-sdk@latest
```

### 2. Update imports in your code

**Before:**
```go
import "github.com/stellar/go/.."
```

**After:**
```go
import "github.com/stellar/go-stellar-sdk/.."
```

### 3. Tidy up
```bash
go mod tidy
```

### 4. SDK Changes

No breaking SDK changes - all functions remain the same.

### 5. Verify
```bash
go list -m github.com/stellar/go
go: module github.com/stellar/go: not a known dependency
```

## Need Help?

Open an issue at: https://github.com/stellar/go-stellar-sdk/issues