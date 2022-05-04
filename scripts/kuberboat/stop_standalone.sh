#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log

# Stop API Server
chmod +x $parent_path/stop_api_server.sh
bash $parent_path/stop_api_server.sh &> /dev/null

# Stop Kubelet
kill -9 `pgrep kubelet` &> /dev/null

# Stop DNS components
$proj_root_path/scripts/dns/stop_dns.sh

# Clear log
rm -rf $log_dir
