# Upgrade Guide

## v23.0.0
This release includes a breaking change to the datastore.

#### 1. The Change
   The ledger file extension has been updated from `.zstd` to the standard `.zst` for Zstandard compression.

#### 2. Datastore Update Options
   You have two options to upgrade Galexie. Both will require you to start fresh and repopulate your entire data lake, as old files are no longer compatible with this version of Galexie. ⚠️ Galexie will refuse to upload files to any `destination_bucket_path` that still contains `.zstd` files.

**Option 1**: Use a New Datastore Location (Recommended)\
    This is the safest method to ensure no issues with the old files. Update the `destination_bucket_path` in your configuration file to a new location which does not contain any ledger files before upgrading Galexie.

**Option 2**: Clean Your Existing Datastore \
   If you want to use your current datastore, you must manually delete all files with the .zstd extension. This will permanently remove your existing data history. Then, restart Galexie.

#### 3. Post-Upgrade Verification
   After restarting, confirm the following:

 - New ledger files are being written with the `.zst` extension.
 - A `.config.json` manifest file has been created in your datastore location (this happens automatically).