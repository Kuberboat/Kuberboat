#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log

# Stop API Server
kill -9 `pgrep apiserver` &> /dev/null

# TODO(wxp): Clear ETCD

# Clear log
rm $log_dir/apiserver.log
