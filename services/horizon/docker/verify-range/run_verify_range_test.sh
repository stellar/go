#!/bin/bash

set -e

: ${JOB_NAME:=test_verify_range$RANDOM}
: ${JOB_QUEUE:=verfy-range-c5-large-queue}
: ${ARRAY_SIZE:=2}
ARRAY_PROPERTIES="size=$ARRAY_SIZE"
: ${JOB_DEFINITION:=verify-range-c5-large-job:4}
: ${BRANCH:=release-horizon-v2.4.0}
: ${BASE_BRANCH:=horizon-v2.0.0}
: ${BATCH_START_LEDGER:=1}
: ${BATCH_SIZE:=30016}
CONTAINER_OVERRIDES="environment=[{name=BRANCH,value=$BRANCH},{name=BASE_BRANCH,value=$BASE_BRANCH},{name=BATCH_START_LEDGER,value=$BATCH_START_LEDGER},{name=BATCH_SIZE,value=$BATCH_SIZE}]"


SUBMISSION_OUTPUT=$(aws batch submit-job --job-name "$JOB_NAME" --job-queue "$JOB_QUEUE"  --job-definition "$JOB_DEFINITION" --array-properties "$ARRAY_PROPERTIES" --container-overrides "$CONTAINER_OVERRIDES")
JOB_ID=$(echo "$SUBMISSION_OUTPUT" | jq -r .jobId)

echo "Job $JOB_NAME launched (Job ID $JOB_ID)"

while true; do
    sleep 15
    STATUS=$(aws batch describe-jobs --jobs "$JOB_ID" | jq .jobs[0].arrayProperties.statusSummary)
    echo "Status:"
    echo "$STATUS"
    FAIL_COUNT=$(echo "$STATUS" | jq .FAILED)
    SUCCEEDED_COUNT=$(echo "$STATUS" | jq .SUCCEEDED)
    if [ "$FAIL_COUNT" != 0 ]; then
	echo "Job failed, exiting"
	exit 1
    fi
    if [ "$SUCCEEDED_COUNT" = "$ARRAY_SIZE" ]; then
	echo "Success"
	exit 0
    fi
done
