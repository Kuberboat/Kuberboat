#!/bin/bash
# set -xe

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
source $parent_path/test_util.sh
log_dir=$proj_root_path/out/log

test_deployment_name="deployment-example"
test_deployment_replica_original=2

create_deployment() {
    $kubectl apply -f $proj_root_path/test/examples/deployment.yaml > /dev/null
    if [ $? -ne 0 ]
        then return -1
    fi
    sleep 25
    if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_deployment_replica_original ]
        then return -1
    fi
}

test_recover() {
    create_deployment
    check_test $? "prepare test deployment"
    pods="$(json_array_value "$($kubectl describe deployment $test_deployment_name)" Pods)"

    echo "kill apiserver"
    kill -9 $(pgrep apiserver)

    sleep 2

    echo "restart apiserver"
    $proj_root_path/out/bin/apiserver &>> $log_dir/apiserver.log &
    if [ -z $(pgrep apiserver) ]; then
        echo "Fail to restart Apiserver"
        exit -1
    fi
    sleep 5
    if [ "$(json_array_value "$($kubectl describe deployment $test_deployment_name)" Pods)" != "$pods" ]; then
        echo "Not consistency"
        exit -1
    fi
}

test_recover
check_test $? "test recover"

clean_test_env