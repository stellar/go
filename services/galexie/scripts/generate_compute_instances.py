#!/usr/bin/python3
"""
This Python script serves as an example of how you could create a series of commands
to create GCP compute instances that run galexie for backfill purposes.

This script may need slight modifications depending on the GCP project
you plan to create compute instances in.
"""

command = """gcloud compute instances create-with-container {instance_name} \
--project={gcp_project} \
--zone={zone} \
--machine-type=e2-standard-2 \
--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default \
--maintenance-policy=MIGRATE \
--provisioning-model=STANDARD \
--service-account={service_account} \
--scopes=https://www.googleapis.com/auth/cloud-platform \
--image=projects/cos-cloud/global/images/cos-stable-113-18244-85-29 \
--boot-disk-size=10GB \
--boot-disk-type=pd-balanced \
--boot-disk-device-name=galexie-pubnet-custom-config \
--container-image=stellar/stellar-galexie:1.0.0 \
--container-restart-policy=always \
--container-privileged \
--container-command=galexie \
--container-arg=append \
--container-arg=--config-file \
--container-arg=/mnt/galexie-config-pubnet/config-pubnet.toml \
--container-arg=--start \
--container-arg={start} \
--container-arg=--end \
--container-arg={end} \
--container-mount-disk=mode=rw,mount-path=/mnt/galexie-config-pubnet,name=galexie-config-pubnet-batch-{batch_num},partition=0 \
--disk=boot=no,device-name=galexie-config-pubnet-batch-{batch_num},mode=rw,name=galexie-config-pubnet-batch-{batch_num},scope=regional \
--no-shielded-secure-boot \
--shielded-vtpm \
--shielded-integrity-monitoring \
--labels=goog-ec-src=vm_add-gcloud,container-vm=cos-stable-113-18244-85-29"""

gcp_project = ""
zone = ""
service_account = ""

commands = []
batch_size = 2500000
start = 0
last_ledger = 52124262

for i in range(1, 22):
    instance_name = f"galexie-pubnet-custom-config-{i}"
    end = start + batch_size - 1
    if i == 21:
        end = last_ledger
    commands.append(command.format(instance_name=instance_name, start=start, end=end, batch_num=i))
    start = end + 1

print(";\n\n".join(commands))
