# Migration Guide

## Migrating from github.com/stellar/go to github.com/stellar/go-stellar-sdk

The Golang SDKs are now located in a new dedicated repo for that sole purpose - https://github.com/stellar/go-stellar-sdk. 

Go client applications that were using SDK packages from `github.com/stellar/go` module will need to change their project to reference the new go module name `github.com/stellar/go-stellar-sdk` and rename package import statements from `github.com/stellar/go/..` to `github.com/stellar/go-stellar-sdk/..`


### 1. Update your go.mod
```bash
# Add the new module name
go get github.com/stellar/go-stellar-sdk@latest

# Remove the old module name
go mod edit -droprequire github.com/stellar/go

go mod tidy
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

### 3. Update git remote (if you have the repo cloned)
```bash
git remote set-url origin https://github.com/stellar/go-stellar-sdk.git
```

### 4. API Changes

No breaking API changes - all functions remain the same.

### 5. Verify
```bash
go mod tidy
go test ./...
```

## Need Help?

Open an issue at: https://github.com/stellar/go-stellar-sdk/issues