#!/bin/bash

echo "========= test fmt ========="

unformatted=`gofmt -l .`

if [ -n "$unformatted" ]
then
    echo "ERROR: unformatted files:"
    echo "$unformatted"
    exit 1
else
    echo "SUCCEESS: test fmt"
fi
