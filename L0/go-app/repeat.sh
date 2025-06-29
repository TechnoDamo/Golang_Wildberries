#!/bin/bash

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <count> <script> [script_args...]"
  exit 1
fi

COUNT=$1
shift
SCRIPT=$1
shift
ARGS="$@"

for ((i = 1; i <= COUNT; i++)); do
  echo "[$i/$COUNT] Running: $SCRIPT $ARGS"
  bash "$SCRIPT" $ARGS
done
