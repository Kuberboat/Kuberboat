#!/bin/bash

mkdir -p $HOME/applog/prometheus

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
target_dir=$parent_path/../../out/config
config_file=$parent_path/prometheus.yml
storage_path=$parent_path/../../out/promdata
log_file=$HOME/applog/prometheus/prometheus.log

mkdir -p $target_dir
prometheus --config.file=$config_file --storage.tsdb.path=$storage_path &>$log_file &

echo "Log file in $log_file"
echo "Please check the log to see if Prometheus starts successfully."
