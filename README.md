# Stellar Go

This repo is the home for all of the go code produced by the stellar organization.

## Layout

In addition to the other top-level packages, there are a few special directories that contain specific types of packages:

* **exp** contains experimental packages.  Use at your own risk.
* **clients** contains packages that provide client packages to the various stellar services.
* **services** contains packages that compile to applications that are long-running processes (such as API servers).
* **tools** contains packages that compile to command line applications.

Each of these directories have their own README file that explain further the nature of their contents.
