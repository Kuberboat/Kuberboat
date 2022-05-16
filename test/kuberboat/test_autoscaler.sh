#!/bin/bash

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

# Test object info. Should be consistent with YAML file.
test_deployment_name="deployment-ubuntu" # deployment_ubuntu.yaml
test_image="ubuntu:20.04"                # deployment_ubuntu.yaml
test_deployment_original_replica=5       # deployment_ubuntu.yaml
test_autoscaler_min_replica=1            # autoscaler.yaml
test_autoscaler_increased_replica=2      # autoscaler.yaml
test_autoscaler_max_replica=3            # autoscaler.yaml

test_create_autoscaler() {
	$kubectl apply -f $proj_root_path/test/examples/deployment_ubuntu.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	$kubectl apply -f $proj_root_path/test/examples/autoscaler.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 60
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) == $test_deployment_original_replica ]; then
		return -1
	fi
}

test_autoscaler_scale_out_by_cpu() {
	container_id=$(docker container ls -q --filter ancestor=$test_image | head -n 1)
	docker exec $container_id /bin/bash -c "cat /dev/urandom | gzip -9 > /dev/null" &
	sleep 60
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_autoscaler_increased_replica ]; then
		return -1
	fi
}

test_autoscaler_scale_in() {
	kill -9 $(pgrep gzip)
	sleep 60
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_autoscaler_min_replica ]; then
		return -1
	fi
}

test_autoscaler_scale_out_by_memory() {
	container_id=$(docker container ls -q --filter ancestor=$test_image | head -n 1)
	docker exec $container_id /bin/bash -c "dd if=/dev/zero of=loadfile bs=1M count=1024" &>/dev/null
	sleep 90
	if [ $(json_digit_value "$($kubectl describe deployment $test_deployment_name)" ReadyReplicas) != $test_autoscaler_max_replica ]; then
		return -1
	fi
}

test_update_deployment_not_allowed() {
	$kubectl apply -f $proj_root_path/test/examples/deployment_ubuntu.yaml &>/dev/null
	if [ $? -eq 0 ]; then
		return -1
	fi
}

test_create_autoscaler
check_test $? "test create autoscaler"

test_autoscaler_scale_out_by_cpu
check_test $? "test autoscaler scale out by cpu"

test_autoscaler_scale_in
check_test $? "test autoscaler scale in"

test_autoscaler_scale_out_by_memory
check_test $? "test autoscaler scale out by memory"

test_update_deployment_not_allowed
check_test $? "test update deployment not allowed"

clean_test_env
