#!/bin/bash
# set -xe

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
log_dir=$proj_root_path/out/log
kubectl=$proj_root_path/out/bin/kubectl
# etcd_ip="192.168.1.13"

start() {
    $parent_path/start_standalone.sh
    tmux new-session -d -s kuberboat-dev "${parent_path}/welcome.sh"
    # If you use local etcd, you may uncomment the following line and replace the IP with you own IP
    # tmux new-window -t kuberboat-dev "etcd -name s1 --initial-advertise-peer-urls http://${etcd_ip}:2380 --listen-peer-urls http://$etcd_ip:2380 --listen-client-urls http://${etcd_ip}:2379,http://127.0.0.1:2379 --advertise-client-urls http://${etcd_ip}:2379 --initial-cluster s1=http://${etcd_ip}:2380 --enable-v2"
    tmux new-window -t kuberboat-dev "tail -f ${log_dir}/kubelet.log"
    tmux new-window -t kuberboat-dev "tail -f ${log_dir}/apiserver.log"
    tmux new-window -t kuberboat-dev "tail -f ~/applog/prometheus/prometheus.log"
    # If you use local etcd, you may comment the following line
    tmux new-window -t kuberboat-dev "docker logs -f etcd"
    sleep 2
    $kubectl apply -f $proj_root_path/test/examples/node.yaml &> /dev/null
}

end() {
    $parent_path/stop_standalone.sh
    tmux kill-session -t kuberboat-dev
}

if [ $# -ne 1 ]; then
    echo "Start and Stop"
elif [ "$1" = 'start' ]; then
    start
elif [ "$1" = 'stop' ]; then
    end
elif [ "$1" = 'restart' ]; then
    end
    start
else
    echo "Unsupported argument."
fi
