# Ledger Exporter (Work in Progress)

The Ledger Exporter is a tool designed to export ledger data from a Stellar network and upload it to a specified destination. It supports both bounded and unbounded modes, allowing users to export a specific range of ledgers or continuously export new ledgers as they arrive on the network.

Ledger Exporter currently uses captive-core as the ledger backend and GCS as the destination data store.

# Exported Data Format
The tool allows for the export of multiple ledgers in a single exported file. The exported data is in XDR format and is compressed using gzip before being uploaded.

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

#### Bounded Mode:
Exports a specific range of ledgers, defined by --start and --end.
```bash
ledgerexporter --start <start_ledger> --end <end_ledger> --config-file <config_file_path>
```

#### Unbounded Mode:
Exports ledgers continuously starting from --start. In this mode, the end ledger is either not provided or set to 0.
```bash
ledgerexporter --start <start_ledger> --config-file <config_file_path>
```

#### Resumability:
Exporting a ledger range can be optimized further by enabling resumability if the remote data store supports it.

By default, resumability is disabled, `--resume false`

When enabled, `--resume true`, ledgerexporter will search the remote data store within the requested range, looking for the oldest absent ledger sequence number within range. If abscence is detected, the export range is narrowed to `--start <absent_ledger_sequence>`. 
This feature requires all ledgers to be present on the remote data store for some (possibly empty) prefix of the requested range and then absent for the (possibly empty) remainder.

### Configuration (toml):

```toml
network = "testnet"  # Options: `testnet` or `pubnet`

[datastore_config]
type = "GCS"

[datastore_config.params]
destination_bucket_path = "your-bucket-name/<optional_subpaths>"

[exporter_config]
ledgers_per_file = 64
files_per_partition = 10
```

#### Stellar-core configuration:
- The exporter automatically configures stellar-core based on the network specified in the config.
- Ensure you have stellar-core installed and accessible in your system's $PATH.

### Exported Files

#### File Organization:
- Ledgers are grouped into files, with the number of ledgers per file set by `ledgers_per_file`.
- Files are further organized into partitions, with the number of files per partition set by `files_per_partition`.

### Filename Structure:
- Filenames indicate the ledger range they contain, e.g., `0-63.xdr.gz` holds ledgers 0 to 63.
- Partition directories group files, e.g., `/0-639/` holds files for ledgers 0 to 639.

#### Example:
with `ledgers_per_file = 64` and `files_per_partition = 10`:
- Partition names: `/0-639`, `/640-1279`, ...
- Filenames: `/0-639/0-63.xdr.gz`, `/0-639/64-127.xdr.gz`, ...

#### Special Cases:

- If `ledgers_per_file` is set to 1, filenames will only contain the ledger number.
- If `files_per_partition` is set to 1, filenames will not contain the partition.

#### Note:
- Avoid changing `ledgers_per_file` and `files_per_partition` after configuration for consistency.

#### Retrieving Data:
- To locate a specific ledger sequence, calculate the partition name and ledger file name using `files_per_partition` and `ledgers_per_file`.
- The `GetObjectKeyFromSequenceNumber` function automates this calculation.

