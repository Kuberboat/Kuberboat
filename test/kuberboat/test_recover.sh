#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
source $parent_path/test_util.sh
source $parent_path/test_deployment.sh
log_dir=$proj_root_path/out/log

test_recover() {
    test_create_deployment
    check_test $? "prepare test deployment"
    pods=$(json_array_value "($kubectl describe $test_deployment_name)" Pods)

    echo "kill apiserver"
    kill -9 $(pgrep apiserver)

    sleep 2

    echo "restart apiserver"
    ./out/bin/apiserver &>> $log_dir/kubelet.log &
    if [ -z $(pgrep apiserver) ]; then
        echo "Fail to restart Apiserver"
        exit -1
    fi
    sleep 5
    if [ $(json_array_value "($kubectl describe $test_deployment_name)" Pods) != pods]; then
        echo "Not consistency"
        exit -1
    fi
}

test_recover

clean_test_env