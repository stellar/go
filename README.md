# Stellar Go

This repo is the home for all of the go code produced by the stellar organization.

## Directory Layout

In addition to the other top-level packages, there are a few special directories that contain specific types of packages:

* **clients** contains packages that provide client packages to the various stellar services.
* **exp** contains experimental packages.  Use at your own risk.
* **handlers** contains packages that provide pluggable implementors of `http.Handler` that make it easier to incorporate portions of the stellar protocol into your own http server. 
* **internal** contains packages that are not intended for consumption outside of stellar's other packages.  Packages that provide common infrastructure for use in our services and tools should go here, such as `db` or `log`. 
* **internal/scripts** contains single-file go programs and bash scripts used to support the development of this repo. 
* **services** contains packages that compile to applications that are long-running processes (such as API servers).
* **tools** contains packages that compile to command line applications.

Each of these directories have their own README file that explain further the nature of their contents.


## Package source layout

While much of the code in individual packages is organized based upon different developers' personal preferences, many of the packages follow a simple convention for organizing the declarations inside of a package that aim to aid in your ability to find code.

Every package should have a `main.go` file.  This file contains the package documentation (unless a separate `doc.go` file is used), _all_ of the exported vars, consts, types and funcs for the package.  It may also contain any unexported declarations that are not tied to any particular type.  In addition to `main.go`, a package often has a single go source file for each types that has method declarations.  This file used the snake case form of the type name (for example `loggly_hook.go` should contain methods for the type `LogglyHook`).

Each non-test file can have a test counterpart like normal, whose name ends with `_test.go`.  Additionally, an `assertions_test.go` file should contain any custom assertions that are related to the package in some way.  This allows a developer to include the package in their tests to gain access to assertions that make writing tests that involve the package more simple.  Finally, a `helpers_test.go` file can contain test utility functions that are not necessarily custom assertions.

In generally, a file is sorted by exported/unexported, then declaration type  (ordered as consts, vars, types, then funcs), then finally alphabetically.