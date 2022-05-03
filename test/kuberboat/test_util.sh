#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
proj_root_path=$parent_path/../..
kubectl=$proj_root_path/out/bin/kubectl
nginx_welcome="Welcome to nginx!"

check_test() {
    if [ $1 -eq 0 ]
    then
        echo "$2 succeeds"
    else
        echo "$2 fails"
        clean_test_env
        exit -1
    fi
}

clean_test_env() {
    $kubectl delete services --all > /dev/null
    sleep 1
    $kubectl delete deployments --all > /dev/null
    sleep 1
    $kubectl delete pods --all > /dev/null
}

json_string_value() {
    regex=".*\"$2\": \"([^\"]*)\""
    [[ $1 =~ $regex ]]
    echo ${BASH_REMATCH[1]}
}

json_digit_value() {
    regex=".*\"$2\": ([0-9]*),?"
    [[ $1 =~ $regex ]]
    echo ${BASH_REMATCH[1]}
}
