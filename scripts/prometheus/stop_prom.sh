#!/bin/bash

kill -9 `pgrep prometheus`

if [ $? -eq 0 ]
    then 
        echo "succesfully stopped prometheus"
        log_file=$HOME/applog/prometheus/prometheus.log
        echo "" > $log_file
    else 
        echo "fail to stop prometheus"
fi
