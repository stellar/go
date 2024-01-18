# Ledger Exporter (Work in Progress)

The Ledger Exporter is a tool designed to export ledger data from a Stellar network and upload it to a specified destination. It supports both bounded and unbounded modes, allowing users to export a specific range of ledgers or continuously export new ledgers as they arrive on the network.

Ledger Exporter currently uses captive-core as the ledger backend and GCS as the destination. 

## Getting Started

### Installation (coming soon)

### Command Line Options

#### Bounded Mode:

```bash
ledgerexporter --start-ledger <start_ledger> --end-ledger <end_ledger> --config-file <config_file_path>
```

#### Unbounded Mode:

```bash
ledgerexporter --start-ledger <start_ledger> --config-file <config_file_path>
```

In unbounded mode, the end ledger is either not provided or set to 0.

## Configuration File (config.toml)

````
network = "testnet"
destination_url = "gcs://your-bucket-name"

[exporter_config]
ledgers_per_file = 64
files_per_partition = 10

[stellar_core_config]
 ...

````


1. **Ledgers Per File:**
  - Determines the range of ledgers included in each file.

2. **Files Per Partition:**
  - Specifies the number of files to be stored within each partition.


The `files_per_partition` configuration parameter works in conjunction with `ledgers_per_file` to also determine the naming of exported files.

### Example

For instance, if `ledgers_per_file` is set to 64 and `files_per_partition` is set to 10, the generated filenames will look like this:

```
/0-639/0-63.xdr    (File 1)
/0-639/64-127.xdr  (File 2)
...
/0-639/576-639.xdr (File 10)
/640-1279/0-63.xdr (File 11, and so on)
...
```

In this example, each directory (`0-639`, `640-1279`, and so on) represents a partition, and within each partition, files are named based on the ledger ranges (`0-63`, `64-127`, and so on), as determined by the `ledgers_per_file` configuration.

**Special Cases:**

- If `ledgers_per_file` is set to 1, filenames will only contain the ledger number.
- If `files_per_partition` is set to 1, filenames will not contain the partition.


**Deterministic File Naming:**

  - Once configured for a particular destination, it is advisable not to change the values of `ledgers_per_file` and `files_per_partition`.
  - The configuration parameters are intended to create deterministic file names. Given a ledger number, users should be able to compute the expected file name based on the configured parameters.
