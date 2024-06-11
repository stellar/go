
# Ledger Exporter Developer Guide
The ledger exporter is a tool to export Stellar network transaction data to cloud storage in a way that is easy to access.

## Prerequisites
This document assumes that you have installed and can run the ledger exporter, and that you have familiarity with its CLI and configuration. If not, please refer to the [Link to Admin Guide]

## Goal
The goal of the ledger exporter is to build an easy-to-use tool to export Stellar network ledger data to cloud storage.
 - Use cloud storage optimally
 - Minimize network usage to export
 - Make it easy and fast to search for a specific ledger or ledger range

## Architecture
To achieve its goals, the ledger exporter uses the following architecture, which consists of the 3 main components:
- Captive-core [Link to existing documentation on captive-core basics] to extract raw transaction metadata from the Stellar Network
- Export manager to bundles and organizes the ledgers to get them ready for export
- The cloud storage plugin writes to the cloud storage. This is specific to the type of cloud storage, GCS in this case.

[Insert Architecture diagram here]

## Data Format
- Ledger exporter uses a compact and efficient data format called XDR (External Data Representation), which is a compact binary format. The captive-core emits data in this format and the data structure is referred to as `LedgerCloseMeta`. The exporter bundle multiple LedgerCloseMeta's into a single object using a custom XDR struct called LedgerCloseMetaBatch which is defined in [Stellar-exporter.x](https://github.com/stellar/go/blob/master/xdr/Stellar-exporter.x).

- The metadata for the same batch is also stored with for each exported object. Supported metadata is defined in [metadata.go](https://github.com/stellar/go/blob/master/support/datastore/metadata.go). 

- Objects are compressed before uploading using the zstd (zstandard) compression algorithm to optimize network usage and storage needs.

## Data Storage (GCS)
- The source for the cloud storage plugin is in the [support](https://github.com/stellar/go/tree/master/support/datastore) package. Any new storage plugin must implement the interface defined in [datastore.go](https://github.com/stellar/go/blob/master/support/datastore/datastore.go). 
- The ledger exporter currently implements the interface only for Google Cloud Storage (GCS). The [GCS plugin](https://github.com/stellar/go/blob/master/support/datastore/gcs_datastore.go) uses GCS-specific behaviors like conditional puts, automatic retry, metadata, and CRC checksum.

## Build, Run and Test using Docker
The Dockerfile contains all the necessary dependencies (e.g., Stellar-core) required to run the ledger exporter. 
- Build: To build the Docker container, use the provided [Makefile](https://github.com/stellar/go/exp/services/ledgerexporter/Makefile). Simply run make `make docker-build` to build a new container after making any changes.

- Run: For instructions on running the Docker container, refer to the Admin Guide [Link to Admin Guide].

- Test: To test the Docker container, refer to the [docker-test](https://github.com/stellar/go/blob/master/exp/services/ledgerexporter/Makefile) command for an example of how to use the [GCS emulator](https://github.com/fsouza/fake-gcs-server) for local testing. 

## Adding support for a new storage type
To add support for a new storage type (e.g. AWS S3), follow these steps:

- Implement the interface defined in [datastore.go](https://github.com/stellar/go/blob/master/support/datastore/datastore.go).
- Add support for new datastore-specific features. Implement any datastore-specific custom logic. Different datastores have different ways of handling 
  - race conditions
  - automatic retries
  - metadata storage, etc.
- Add the new datastore to the factory function [NewDataStore](https://github.com/stellar/go/blob/master/support/datastore/datastore.go).
- Add a [config](https://github.com/stellar/go/blob/master/exp/services/ledgerexporter/config.toml) section for the new storage type. This includes configurations like destination, authentication information etc.
- An emulator such as a GCS emulator [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) can be used for testing without connecting to real cloud storage.

### Design DOs and DONTs
- Multiple exporters should be able to run in parallel without the need for explicit locking or synchronization.
- Exporters when restarted do not have any memory of prior operation and rely on the already exported data as much as possible to decide where to resume.

## Using exported data
The exported data in storage can be used in the ETL pipeline to gather analytics and reporting. To write a tool that consumes exported data you can use Stellar ingestion library's `ledgerbackend` package. This package includes a ledger backend called [BufferedStorageBackend](https://github.com/stellar/go/blob/master/ingest/ledgerbackend/buffered_storage_backend.go),
which imports data from the storage and validates it. For more details, refer to the ledgerbackend [documentation](https://github.com/stellar/go/tree/master/ingest/ledgerbackend).

## Contributing
For information on how to contribute, please refer to our [Contribution Guidelines](https://github.com/stellar/go/blob/master/CONTRIBUTING.md).
