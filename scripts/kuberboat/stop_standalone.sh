#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log

# Stop API Server
chmod +x $parent_path/stop_api_server.sh
bash $parent_path/stop_api_server.sh &> /dev/null

# Stop Kubelet
kill -9 `pgrep kubelet` &> /dev/null

# Stop API Server
kill -9 `pgrep apiserver` &> /dev/null

# Clear etcd.
chmod +x $proj_root_path/scripts/kuberboat/clear_etcd.sh
bash $proj_root_path/scripts/kuberboat/clear_etcd.sh

# Stop DNS components
chmod +x $proj_root_path/scripts/dns/stop_dns.sh
bash $proj_root_path/scripts/dns/stop_dns.sh

# Clear log
rm -rf $log_dir
