#!/bin/bash

# set -xe

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

# Test object info. Should be consistent with YAML file.
test_deployment_name="deployment-example" # deployment.yaml
test_deployment_replica_original=2        # deployment.yaml
test_deployment_replica_changed=4         # deployment_replica_change.yaml

test_create_deployment() {
	$kubectl apply -f $proj_root_path/test/examples/deployment.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 25
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_deployment_replica_original ]; then
		return -1
	fi
}

test_deployment_maintain_replica_1() {
	$kubectl delete pods --all >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != 0 ]; then
		return -1
	fi
	sleep 25
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_deployment_replica_original ]; then
		return -1
	fi
}

test_deployment_maintain_replica_2() {
	$kubectl apply -f $proj_root_path/test/examples/deployment_replica_change.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" Replicas) != $test_deployment_replica_changed ]; then
		return -1
	fi
	sleep 25
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_deployment_replica_changed ]; then
		return -1
	fi
}

test_delete_deployment() {
	$kubectl delete deployment $test_deployment_name >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 2
	grep -q "deployments are not found" <<<$($kubectl describe deployment $test_deployment_name)
	return $?
}

test_create_deployment
check_test $? "test create deployment"

test_deployment_maintain_replica_1
check_test $? "test deployment maintain replica 1"

test_deployment_maintain_replica_2
check_test $? "test deployment maintain replica 2"

test_delete_deployment
check_test $? "test delete deployment"

clean_test_env
