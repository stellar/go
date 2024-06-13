# Ledger Exporter (Work in Progress)

The Ledger Exporter is a tool designed to export ledger data from a Stellar network and upload it to a specified destination. It supports both bounded and unbounded modes, allowing users to export a specific range of ledgers or continuously export new ledgers as they arrive on the network.

Ledger Exporter currently uses captive-core as the ledger backend and GCS as the destination data store.

# Exported Data Format
The tool allows for the export of multiple ledgers in a single exported file. The exported data is in XDR format and is compressed using zstd before being uploaded.

```go
type LedgerCloseMetaBatch struct {
    StartSequence uint32
    EndSequence uint32
    LedgerCloseMetas []LedgerCloseMeta
}
```

## Getting Started

### Installation (coming soon)

### Command Line Options

#### Scan and Fill Mode:
Exports a specific range of ledgers, defined by --start and --end. Will only export to remote datastore if data is absent.
```bash
ledgerexporter scan-and-fill --start <start_ledger> --end <end_ledger> --config-file <config_file_path>
```

#### Append Mode:
Exports ledgers initially searching from --start, looking for the next absent ledger sequence number proceeding --start on the data store. If abscence is detected, the export range is narrowed to `--start <absent_ledger_sequence>`. 
This feature requires ledgers to be present on the remote data store for some (possibly empty) prefix of the requested range and then absent for the (possibly empty) remainder. 

In this mode, the --end ledger can be provided to stop the process once export has reached that ledger, or if absent or 0 it will result in continous exporting of new ledgers emitted from the network. 

 It’s guaranteed that ledgers exported during `append` mode from `start` and up to the last logged ledger file `Uploaded {ledger file name}` were contiguous, meaning all ledgers within that range were exported to the data lake with no gaps or missing ledgers in between.
```bash
ledgerexporter append --start <start_ledger> --config-file <config_file_path>
```

### Configuration (toml):
The `stellar_core_config` supports two ways for configuring captive core:
  - use prebuilt captive core config toml, archive urls, and passphrase based on `stellar_core_config.network = testnet|pubnet`.
  - manually set the the captive core confg by supplying these core parameters which will override any defaults when `stellar_core_config.network` is present also:
    `stellar_core_config.captive_core_toml_path`
    `stellar_core_config.history_archive_urls`
    `stellar_core_config.network_passphrase`

Ensure you have stellar-core installed and set `stellar_core_config.stellar_core_binary_path` to it's path on o/s.

Enable web service that will be bound to localhost post and publishes metrics by including `admin_port = {port}`

An example config, demonstrating preconfigured captive core settings and gcs data store config.
```toml
admin_port = 6061

[datastore_config]
type = "GCS"

[datastore_config.params]
destination_bucket_path = "your-bucket-name/<optional_subpath1>/<optional_subpath2>/"

[datastore_config.schema]
ledgers_per_file = 64
files_per_partition = 10

[stellar_core_config]
  network = "testnet"
  stellar_core_binary_path = "/my/path/to/stellar-core"
  captive_core_toml_path = "my-captive-core.cfg"
  history_archive_urls = ["http://testarchiveurl1", "http://testarchiveurl2"]
  network_passphrase = "test"
```

### Exported Files

#### File Organization:
- Ledgers are grouped into files, with the number of ledgers per file set by `ledgers_per_file`.
- Files are further organized into partitions, with the number of files per partition set by `files_per_partition`.

### Filename Structure:
- Filenames indicate the ledger range they contain, e.g., `0-63.xdr.zstd` holds ledgers 0 to 63.
- Partition directories group files, e.g., `/0-639/` holds files for ledgers 0 to 639.

#### Example:
with `ledgers_per_file = 64` and `files_per_partition = 10`:
- Partition names: `/0-639`, `/640-1279`, ...
- Filenames: `/0-639/0-63.xdr.zstd`, `/0-639/64-127.xdr.zstd`, ...

#### Special Cases:

- If `ledgers_per_file` is set to 1, filenames will only contain the ledger number.
- If `files_per_partition` is set to 1, filenames will not contain the partition.

#### Note:
- Avoid changing `ledgers_per_file` and `files_per_partition` after configuration for consistency.

#### Retrieving Data:
- To locate a specific ledger sequence, calculate the partition name and ledger file name using `files_per_partition` and `ledgers_per_file`.
- The `GetObjectKeyFromSequenceNumber` function automates this calculation.

