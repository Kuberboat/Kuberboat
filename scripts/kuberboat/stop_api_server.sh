#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log

# Stop API Server
kill -9 `pgrep apiserver` &> /dev/null

api_objects=("/Pods" "/Deployments" "/Services" "/Nodes")

for i in "${api_objects[@]}"; do
    etcdctl del $i --prefix
done

# Clear log
rm $log_dir/apiserver.log
