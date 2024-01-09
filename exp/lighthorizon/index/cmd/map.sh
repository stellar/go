#!/bin/bash
# 
# Breaks up the given ledger dumps into checkpoints and runs a map
# job on each one. However, it's the Golang side does validation that 
# the map job resulted in the correct indices.
# 

# check parameters and their validity (types, existence, etc.)

if [[ "$#" -ne "2" ]]; then 
    echo "Usage: $0 <txmeta src> <index dest>"
    exit 1
fi

if [[ ! -d "$1" ]]; then 
    echo "Error: txmeta src ('$1') does not exist"
    echo "Usage: $0 <txmeta src> <index dest>"
    exit 1
fi

if [[ -z $BATCH_SIZE ]]; then 
    echo "BATCH_SIZE environmental variable required"
    exit 1
elif ! [[ $BATCH_SIZE =~ ^[0-9]+$ ]]; then 
    echo "BATCH_SIZE ('$BATCH_SIZE') must be an integer"
    exit 1
fi

if [[ -z $FIRST_LEDGER || -z $LAST_LEDGER ]]; then 
    echo "FIRST_LEDGER and LAST_LEDGER environmental variables required"
    exit 1
elif ! [[ $FIRST_LEDGER =~ ^[0-9]+$ && $LAST_LEDGER =~ ^[0-9]+$ ]]; then 
    echo "FIRST_LEDGER ('$FIRST_LEDGER') and LAST_LEDGER ('$LAST_LEDGER') must be integers"
    exit 1
fi

if [[ ! -d "$2" ]]; then 
    echo "Warning: index dest ('$2') does not exist, creating..."
    mkdir -p $2
fi

# do work

FIRST=$FIRST_LEDGER
LAST=$LAST_LEDGER
COUNT=$(($LAST-$FIRST+1))
# batches = ceil(count / batch_size)
# formula is from https://stackoverflow.com/a/12536521
BATCH_COUNT=$(( ($COUNT + $BATCH_SIZE - 1) / $BATCH_SIZE ))

if [[ "$(((LAST + 1) % 64))" -ne "0" ]]; then
    echo "LAST_LEDGER ($LAST_LEDGER) should be a checkpoint ledger"
    exit 1
fi

echo " - start: $FIRST"
echo " - end:   $LAST"
echo " - count: $COUNT ($BATCH_COUNT batches @ $BATCH_SIZE ledgers each)"

go build -o ./map ./batch/map/...
if [[ "$?" -ne "0" ]]; then 
    echo "Build failed"
    exit 1
fi

pids=( )
for (( i=0; i < $BATCH_COUNT; i++ ))
do
    echo -n "Creating map job $i... "

    NETWORK_PASSPHRASE='testnet' JOB_INDEX_ENV='AWS_BATCH_JOB_ARRAY_INDEX' MODULES='accounts_unbacked,transactions' \
    AWS_BATCH_JOB_ARRAY_INDEX=$i BATCH_SIZE=$BATCH_SIZE FIRST_CHECKPOINT=$FIRST \
    TXMETA_SOURCE=file://$1 INDEX_TARGET=file://$2 WORKER_COUNT=1 \
        ./map &
    
    echo "pid=$!"
    pids+=($!)
done

sleep $BATCH_COUNT

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

rm ./map
echo "All jobs succeeded!"
exit 0
