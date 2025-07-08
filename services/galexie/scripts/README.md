## Galexie: Backfill Examples

The files in this directory are examples in different ways to use Galexie to backfill Stellar network data into a Google Cloud Storage (GCS) bucket.

## Notes and Tips

* An unoptimized full history backfill with pubnet data using Galexie took roughly 4.5 days
* Total costs ~= $1100 USD
  * Compute Costs ~= $500 USD
  * GCS Class A Operations (writes) Costs ~= $600 USD
* Pubnet full history size is ~= 3 TB (as of 2024-07-31)
* Using Galexie for earlier ledgers will be processed faster than ledgers closer to the current time. This is due to the fact that ledgers closer to the current time have more data due to additional features added over the years as well as larger adoption and usage of the Stellar network in general.
* There is a noticable inflection point in runtime around ledger 30000000 (30 million). At this time it is recommened to use smaller ledger ranges for the backfilling process.
* There are extra flags that can be enabled in the captive-core.cfg to output extra information such as `ENABLE_SOROBAN_DIAGNOSTIC_EVENTS`. Please see more captive-core options [here](https://github.com/stellar/go/blob/f692f1246b01fb09af2c232630d4ad31025de747/ingest/ledgerbackend/toml.go#L74-L109)
* Large ledger ranges (e.g., 100000 VS 2500000 ledger range) may slow down processing speed (this assumption has not been confirmed and may not affect your use case)

## Instructions for generate_compute_instance.py

* This Python script will generate `gcloud compute instance` commands that you can run in your terminal/shell to create compute instances that run Galexie over a specified ledger range
* To use this script please fill out the variables between lines 41 and 48. This will include information such as your GCP project, zone, and service account you wish to use to execute Galexie with.
* You will need to create the volume/disk mounts that contain the Galexie and captive core configuration files

```
--container-mount-disk=mode=rw,mount-path=/mnt/galexie-config-pubnet,name=galexie-config-pubnet-batch-{batch_num},partition=0 \
```

* Note that these compute instances will not spin down on their own. The Galexie image will complete and will be stuck in an infinite retry loop. Please manually stop the compute instance when all ledgers for the ledger range have been written

## Instructions for batch_config.yml

* This YAML file is a job configuration that creates compute instances to run Galexie using [GCP batch](https://cloud.google.com/batch)
* This will not run as is and will need users to modify the tasks as well as the mount disks containing the Galexie and captive core configuration files
* This file can be used like so

```
gcloud batch jobs submit galexie-batch --config batch_config.yml
```
