#!/bin/bash

if [ -z $1 ]; then
  echo "invalid argument: must specify a data set ID."
  exit 1;
fi

curl -X POST "http://localhost:4917/api/v1/data/$1/upload"

