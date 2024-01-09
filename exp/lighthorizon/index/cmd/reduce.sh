#!/usr/bin/env bash
# 
# Combines indices that were built separately in different folders into a single
# set of indices.
# 
# This focuses on starting parallel processes, but the Golang side does
# validation that the reduce jobs resulted in the correct indices.
# 

# check parameters and their validity (types, existence, etc.)

if [[ "$#" -ne "2" ]]; then 
    echo "Usage: $0 <index src root> <index dest>"
    exit 1
fi

if [[ ! -d "$1" ]]; then 
    echo "Error: index src root ('$1') does not exist"
    echo "Usage: $0 <index src root> <index dest>"
    exit 1
fi

if [[ ! -d "$2" ]]; then 
    echo "Warning: index dest ('$2') does not exist, creating..."
    mkdir -p "$2"
fi

MAP_JOB_COUNT=$(ls $1 | grep -E 'job_[0-9]+' | wc -l)
if [[ "$MAP_JOB_COUNT" -le "0" ]]; then 
    echo "No jobs in index src root ('$1') found."
    exit 1 
fi
REDUCE_JOB_COUNT=$MAP_JOB_COUNT

# build reduce program and start it up

go build -o reduce ./batch/reduce/...
if [[ "$?" -ne "0" ]]; then 
    echo "Build failed"
    exit 1
fi

echo "Coalescing $MAP_JOB_COUNT discovered job outputs from $1 into $2..."

pids=( )
for (( i=0; i < $REDUCE_JOB_COUNT; i++ ))
do
    echo -n "Creating reduce job $i... "

    AWS_BATCH_JOB_ARRAY_INDEX=$i JOB_INDEX_ENV="AWS_BATCH_JOB_ARRAY_INDEX" MAP_JOB_COUNT=$MAP_JOB_COUNT \
    REDUCE_JOB_COUNT=$REDUCE_JOB_COUNT WORKER_COUNT=4 \
    INDEX_SOURCE_ROOT=file://$1 INDEX_TARGET=file://$2 \
        timeout -k 30s 10s ./reduce &

    echo "pid=$!"
    pids+=($!)
done

sleep $REDUCE_JOB_COUNT

# Check the status codes for all of the map processes.
for i in "${!pids[@]}"; do
    pid=${pids[$i]}
    echo -n "Checking job $i (pid=$pid)... "
    if ! wait "$pid"; then
        echo "failed"
        exit 1
    else
        echo "succeeded!"
    fi
done

rm ./reduce # cleanup
echo "All jobs succeeded!"
exit 0
