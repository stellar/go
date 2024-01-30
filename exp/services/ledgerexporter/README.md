# Ledger Exporter (Work in Progress)

The Ledger Exporter is a tool designed to export ledger data from a Stellar network and upload it to a specified destination. It supports both bounded and unbounded modes, allowing users to export a specific range of ledgers or continuously export new ledgers as they arrive on the network.

Ledger Exporter currently uses captive-core as the ledger backend and GCS as the destination data store.

# Exported Data Format
The tool allows for the export of multiple ledgers in a single exported file. The exported data is in XDR format and is compressed (gzip) before being uploaded.

```bash
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


Starts exporting from a specified number of ledgers before the latest ledger sequence number on the network.
```bash
ledgerexporter --from-last <number_of_ledgers> --config-file <config_file_path>
```


## Configuration File (config.toml)

````
network = "testnet"
destination_url = "gcs://your-bucket-name"

[exporter_config]
ledgers_per_file = 64
files_per_partition = 10

[stellar_core_config]
  captive_core_toml_path = <path-to-captive-core-config-toml>
  network_passphrase = "Test SDF Network ; September 2015"
  history_archive_urls = <history-archive-urls>
  stellar_core_binary_path = <stellar-core-binary-path>
  captive_core_use_db = true

````


1. **Ledgers Per File:**
  - Determines the range of ledgers included in each file.

2. **Files Per Partition:**
  - Specifies the number of files to be stored within each partition.


The `files_per_partition` configuration parameter works in conjunction with `ledgers_per_file` to also determine the naming of exported files.

### Exported Filenames

For instance, if `ledgers_per_file` is set to 64 and `files_per_partition` is set to 10, the generated filenames will look like this:

```
/0-639/0-63.xdr.gz (File 1)
/0-639/64-127.xdr  (File 2)
...
/0-639/576-639.xdr.gz (File 10)
/640-1279/0-63.xdr.gz (File 11, and so on)
...
```

In this example, each directory (`0-639`, `640-1279`, and so on) represents a partition, and within each partition, files are named based on the ledger ranges (`0-63`, `64-127`, and so on), as determined by the `ledgers_per_file` configuration.

**Special Cases:**

- If `ledgers_per_file` is set to 1, filenames will only contain the ledger number.
- If `files_per_partition` is set to 1, filenames will not contain the partition.


**Deterministic File Naming:**

  - Once configured for a particular destination, it is advisable not to change the values of `ledgers_per_file` and `files_per_partition`.
  - The configuration parameters are intended to create deterministic file names. Given a ledger number, users should be able to compute the expected file name based on the configured parameters.
