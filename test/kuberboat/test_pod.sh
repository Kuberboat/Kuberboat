#!/bin/bash

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

# Test object info. Should be consistent with YAML file.
test_pod_name="mypod0" # pod.yaml

test_create_pod() {
	$kubectl apply -f $proj_root_path/test/examples/pod.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 20
	if [ $(json_string_value "$($kubectl describe pod $test_pod_name)" Phase) != "Ready" ]; then
		return -1
	fi
}

test_delete_pod() {
	$kubectl delete pod $test_pod_name >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 2
	grep -q "pods are not found" <<<$($kubectl describe pod $test_pod_name)
	return $?
}

test_create_pod
check_test $? "test create pod"

test_delete_pod
check_test $? "test delete pod"

clean_test_env
