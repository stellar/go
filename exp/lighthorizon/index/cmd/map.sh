#!/bin/bash
# 
# Breaks up the given ledger dumps into checkpoints and runs a map
# job on each one. However, it's the Golang side does validation that 
# the map job resulted in the correct indices.
# 

if [[ "$#" -ne "2" ]]; then 
    echo "Usage: $0 <txmeta src> <index dest>"
    exit 1
fi

if [[ ! -d "$1" ]]; then 
    echo "Error: txmeta src ('$1') does not exist"
    echo "Usage: $0 <txmeta src> <index dest>"
    exit 1
fi

if [[ ! -d "$2" ]]; then 
    echo "Warning: index dest ('$2') does not exist, creating..."
    mkdir -p $2
fi

echo "Analyzing $1:"
LATEST=$(cat $1/latest)
LAST=$(ls $1/ledgers | tail -n1)

if [[ "$LATEST" -ne "$LAST" ]]; then 
    echo "Latest ledger incorrect: $LAST is last but $LATEST reported"
    exit 1
fi

FIRST=$(ls $1/ledgers | head -n1)
COUNT=$(($LAST-$FIRST))
CHECKPOINT_COUNT=$(($COUNT / 64))

echo " - start: $FIRST"
echo " - end:   $LAST"
echo " - count: $COUNT ($CHECKPOINT_COUNT checkpoints)"

if [[ "$((($FIRST + 1) % 64))" -ne "0" ]]; then 
    echo "$FIRST isn't a checkpoint ledger, adjusting..."
    exit 1
fi

go build -o ./map ./batch/map/...
if [[ "$?" -ne "0" ]]; then 
    echo "Build failed"
    exit 1
fi

# Because for i in {0..$CHECKPOINT_COUNT} won't work...
# https://www.cyberciti.biz/faq/unix-linux-iterate-over-a-variable-range-of-numbers-in-bash/
pids=( )
for i in $(eval echo "{0..$((CHECKPOINT_COUNT-1))}")
do
    echo "Creating job $i"

    AWS_BATCH_JOB_ARRAY_INDEX=$i BATCH_SIZE=64 FIRST_CHECKPOINT=$FIRST \
    TXMETA_SOURCE=file://$1 INDEX_TARGET=file://$2 WORKER_COUNT=2 \
        ./map & pids+=( $! )
done

# Check the status codes for all of the map processes.
for i in "${!pids[@]}"; do
  pid=${pids[$i]}
  echo "Checking job $i: pid=$pid"
  if ! wait "$pid"; then
    echo "failed"
    exit 1
  fi
done

rm ./map
echo "All jobs succeeded!"
exit 0