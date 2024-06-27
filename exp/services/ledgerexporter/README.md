## Ledger Exporter: Installation and Usage Guide

This guide provides step-by-step instructions on installing and using the Ledger Exporter, a tool that exports Stellar network ledger data to a Google Cloud Storage (GCS) bucket for efficient analysis and storage.

* [Prerequisites](#prerequisites)
* [Setup](#setup)
  * [Set Up GCP Credentials](#set-up-gcp-credentials)
  * [Create a GCS Bucket for Storage](#create-a-gcs-bucket-for-storage)
* [Running the Ledger Exporter](#running-the-ledger-exporter)
  * [Pull the Docker Image](#1-pull-the-docker-image)
  * [Configure the Exporter](#2-configure-the-exporter-configtoml)
  * [Run the Exporter](#3-run-the-exporter)
* [Command Line Interface (CLI)](#command-line-interface-cli)
  1. [scan-and-fill: Fill Data Gaps](#1-scan-and-fill-fill-data-gaps)
  2. [append: Continuously Export New Data](#2-append-continuously-export-new-data)

## Prerequisites

* **Google Cloud Platform (GCP) Account:**  You will need a GCP account to create a GCS bucket for storing the exported data.
* **Docker:** Allows you to run the Ledger Exporter in a self-contained environment. The official Docker installation guide: [https://docs.docker.com/engine/install/](https://docs.docker.com/engine/install/)

## Setup

### Set Up GCP Credentials

Create application default credentials for your Google Cloud Platform (GCP) project by following these steps:
1. Download the [SDK](https://cloud.google.com/sdk/docs/install).
2. Install and initialize the [gcloud CLI](https://cloud.google.com/sdk/docs/initializing).
3. Create [application authentication credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc#google-idp) and store it in a secure location on your system, such as $HOME/.config/gcloud/application_default_credentials.json.

For detailed instructions, refer to the [Providing Credentials for Application Default Credentials (ADC) guide.](https://cloud.google.com/docs/authentication/provide-credentials-adc)

### Create a GCS Bucket for Storage

1. Go to the GCP Console's Storage section ([https://console.cloud.google.com/storage](https://console.cloud.google.com/storage)) and create a new bucket.
2. Choose a descriptive name for the bucket, such as `stellar-ledger-data`. Refer to [Google Cloud Storage Bucket Naming Guideline](https://cloud.google.com/storage/docs/buckets#naming) for more information.
3. **Note down the bucket name** as you'll need it later in the configuration process.


## Running the Ledger Exporter

### 1. Pull the Docker Image

Open a terminal window and download the Stellar Ledger Exporter Docker image using the following command:

```bash
docker pull stellar/ledger-exporter
```

### 2. Configure the Exporter (config.toml)
The Ledger Exporter relies on a configuration file (config.toml) to connect to your specific environment. This file defines details like:
- Your Google Cloud Storage (GCS) bucket where exported ledger data will be stored.
- Stellar network settings, such as the network you're using (testnet or pubnet).
- Datastore schema to control data organization.

A sample configuration file [config.example.toml](config.example.toml) is provided. Copy and rename it to config.toml for customization. Edit the copied file (config.toml) to replace placeholders with your specific details.

### 3. Run the Exporter

The following command demonstrates how to run the Ledger Exporter:

```bash
docker run --platform linux/amd64 \
  -v "$HOME/.config/gcloud/application_default_credentials.json":/.config/gcp/credentials.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/.config/gcp/credentials.json \
  -v ${PWD}/config.toml:/config.toml \
  stellar/ledger-exporter <command> [options]
```

**Explanation:**

* `--platform linux/amd64`: Specifies the platform architecture (adjust if needed for your system).
* `-v`: Mounts volumes to map your local GCP credentials and config.toml file to the container:
  * `$HOME/.config/gcloud/application_default_credentials.json`: Your local GCP credentials file.
  * `${PWD}/config.toml`: Your local configuration file.
* `-e GOOGLE_APPLICATION_CREDENTIALS=/.config/gcp/credentials.json`: Sets the environment variable for credentials within the container.
* `stellar/ledger-exporter`: The Docker image name.
* `<command>`: The Stellar Ledger Exporter command: [append](#1-append-continuously-export-new-data), [scan-and-fill](#2-scan-and-fill-fill-data-gaps))

## Command Line Interface (CLI)

The Ledger Exporter offers two mode of operation for exporting ledger data:

### 1. append: Continuously Export New Data


Exports ledgers initially searching from --start, looking for the next absent ledger sequence number proceeding --start on the data store. If abscence is detected, the export range is narrowed to `--start <absent_ledger_sequence>`. 
This feature requires ledgers to be present on the remote data store for some (possibly empty) prefix of the requested range and then absent for the (possibly empty) remainder. 

In this mode, the --end ledger can be provided to stop the process once export has reached that ledger, or if absent or 0 it will result in continous exporting of new ledgers emitted from the network. 

It’s guaranteed that ledgers exported during `append` mode from `start` and up to the last logged ledger file `Uploaded {ledger file name}` were contiguous, meaning all ledgers within that range were exported to the data lake with no gaps or missing ledgers in between.


**Usage:**

```bash
docker run --platform linux/amd64 -d \
  -v "$HOME/.config/gcloud/application_default_credentials.json":/.config/gcp/credentials.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/.config/gcp/credentials.json \
  -v ${PWD}/config.toml:/config.toml \
  stellar/ledger-exporter \
  append --start <start_ledger> [--end <end_ledger>] [--config-file <config_file>]
```

Arguments:
- `--start <start_ledger>` (required): The starting ledger sequence number for the export process.
- `--end <end_ledger>` (optional): The ending ledger sequence number. If omitted or set to 0, the exporter will continuously export new ledgers as they appear on the network.
- `--config-file <config_file_path>` (optional): The path to your configuration file, containing details like GCS bucket information. If not provided, the exporter will look for config.toml in the directory where you run the command.

### 2. scan-and-fill: Fill Data Gaps

Scans the datastore (GCS bucket) for the specified ledger range and exports any missing ledgers to the datastore. This mode avoids unnecessary exports if the data is already present. The range is specified using the --start and --end options.

**Usage:**

```bash
docker run --platform linux/amd64 -d \
  -v "$HOME/.config/gcloud/application_default_credentials.json":/.config/gcp/credentials.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/.config/gcp/credentials.json \
  -v ${PWD}/config.toml:/config.toml \
  stellar/ledger-exporter \
  scan-and-fill --start <start_ledger> --end <end_ledger> [--config-file <config_file>]
```

Arguments:
- `--start <start_ledger>` (required): The starting ledger sequence number in the range to export.
- `--end <end_ledger>` (required): The ending ledger sequence number in the range.
- `--config-file <config_file_path>` (optional): The path to your configuration file, containing details like GCS bucket information. If not provided, the exporter will look for config.toml in the directory where you run the command.

