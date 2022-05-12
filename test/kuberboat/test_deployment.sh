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
test_rolling_update_replicas=5            # deployment_v2.yaml
test_rolling_update_version=v2            # deployment_v2.yaml

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

test_rolling_update() {
	$kubectl apply -f $proj_root_path/test/examples/deployment_v2.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 30
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" Replicas) != $test_rolling_update_replicas ]; then
		return -1
	fi
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_rolling_update_replicas ]; then
		return -1
	fi
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" UpdatedReplicas) != $test_rolling_update_replicas ]; then
		return -1
	fi
	if [ $(json_string_value "$($kubectl describe deployment $test_deployment_name)" version) != $test_rolling_update_version ]; then
		return -1
	fi
}

test_delete_deployment() {
	$kubectl delete deployment $test_deployment_name >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5
	grep -q "deployments are not found" <<<$($kubectl describe deployment $test_deployment_name)
	return $?
}

test_create_deployment
check_test $? "test create deployment"

test_deployment_maintain_replica_1
check_test $? "test deployment maintain replica 1"

test_deployment_maintain_replica_2
check_test $? "test deployment maintain replica 2"

test_rolling_update
check_test $? "test rolling update"

test_delete_deployment
check_test $? "test delete deployment"

clean_test_env
