#!/bin/bash
set -e

# Parse command line arguments
START_TIME=$1
END_TIME=$2
OUTPUT_DIR=$3

if [ -z "$START_TIME" ] || [ -z "$END_TIME" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "Usage: $0 <start-time> <end-time> <output-directory>"
    echo "Example: $0 2021-01-01T00:00:00+00:00 2021-01-01T00:01:00+00:00 /path/to/output"
    exit 1
fi

# Create output directory and errors subdirectory
mkdir -p "$OUTPUT_DIR/ERRORS"

# Extract date from start time for naming files
DATE_PREFIX=$(echo "$START_TIME" | cut -d'T' -f1)
RANGE_FILE="${OUTPUT_DIR}/${DATE_PREFIX}-exported_range.txt"
BAD_LEDGERS_FILE="${OUTPUT_DIR}/${DATE_PREFIX}-bad-ledgers.txt"

echo "Getting ledger range for time period: $START_TIME to $END_TIME"
stellar-etl get_ledger_range_from_times \
    --start-time "$START_TIME" \
    --end-time "$END_TIME" \
    --output "$RANGE_FILE"

if [ ! -f "$RANGE_FILE" ]; then
    echo "Error: Ledger range file was not created. Exiting."
    exit 1
fi

# Parse the JSON to extract start and end ledgers
START_LEDGER=$(cat "$RANGE_FILE" | jq -r '.start')
END_LEDGER=$(cat "$RANGE_FILE" | jq -r '.end')

if [ -z "$START_LEDGER" ] || [ -z "$END_LEDGER" ]; then
    echo "Error: Failed to extract ledger range values."
    exit 1
fi

echo "Processing ledger range: $START_LEDGER to $END_LEDGER"

# Run the Go program with the extracted ledger range
/tmp/bsb_ttp \
    --start-ledger "$START_LEDGER" \
    --end-ledger "$END_LEDGER" \
    --output-dir "$OUTPUT_DIR" \
    --errors-dir "$OUTPUT_DIR/ERRORS" \
    --bad-ledgers-file "$BAD_LEDGERS_FILE" \
    --date-prefix "$DATE_PREFIX"

echo "Processing complete. Check for errors in $OUTPUT_DIR/ERRORS/ and bad ledgers in $BAD_LEDGERS_FILE"
