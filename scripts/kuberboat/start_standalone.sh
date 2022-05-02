#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log
prometheus_dir=$proj_root_path/scripts/prometheus

# Clear old states
kill -9 `pgrep apiserver` &> /dev/null
kill -9 `pgrep kubelet` &> /dev/null

# Build
cd $proj_root_path && make > /dev/null
if [ $? -ne 0 ]
then 
    echo "Fail to build the project"
    exit -1
else 
    echo "Project successfully built"
fi

# Start Prometheus
if [ -z `pgrep prometheus` ]
then 
    chmod +x $prometheus_dir/start_prom.sh
    $prometheus_dir/start_prom.sh
else
    echo "Prometheus alreay started"
fi

# Start ETCD
docker start etcd &> /dev/null
if [ $? -ne 0 ]
then
    mkdir -p /tmp/etcd-data.tmp
    docker run -d \
        --name etcd \
        -p 2379:2379 \
        --restart always \
        --mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data \
        quay.io/coreos/etcd:v3.5.2 \
        /usr/local/bin/etcd \
        --name s1 \
        --data-dir /etcd-data \
        --listen-client-urls http://0.0.0.0:2379 \
        --advertise-client-urls http://0.0.0.0:2379 \
        --listen-peer-urls http://0.0.0.0:2380 \
        --initial-advertise-peer-urls http://0.0.0.0:2380 \
        --initial-cluster s1=http://0.0.0.0:2380 \
        --initial-cluster-token tkn \
        --initial-cluster-state new \
        --enable-v2 > /dev/null
    if [ $? -eq 0 ]
    then
        echo "ETCD successfully started"
    else
        echo "Fail to start ETCD"
    fi
else
    echo "ETCD already started"
fi

# Create log directory
mkdir -p $log_dir

# Start API Server
./out/bin/apiserver &> $log_dir/apiserver.log &

# Start Kubelet
./out/bin/kubelet &> $log_dir/kubelet.log &

# Wait for API Server and Kubelet to fully start
sleep 1.5
declare -i exit_state
exit_state=0

# Check whether API Server starts successfully
if [ -z `pgrep apiserver` ]
then 
    echo "Fail to start API Server"
    exit_state=-1
else 
    echo "API Server successfully started"
fi

# Check whether Kubelet starts successfully
if [ -z `pgrep kubelet` ]
then 
    echo "Fail to start Kubelet"
    exit_state=-1
else 
    echo "Kubelet successfully started"
fi

exit $exit_state
