#!/bin/bash

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

# Test object info. Should be consistent with YAML file.
test_pod_1_name="test-pod-1"      # pod_nginx_1.yaml
test_pod_2_name="test-pod-2"      # pod_nginx_2.yaml
test_pod_3_name="test-pod-3"      # pod_nginx_3.yaml
test_container_name="nginx"       # pod_nginx_1.yaml, pod_nginx_2.yaml, pod_nginx_3.yaml
test_service_name="nginx-service" # service.yaml
test_service_port=8088            # service.yaml

test_create_service() {
	$kubectl apply -f $proj_root_path/test/examples/pod_nginx_1.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	$kubectl apply -f $proj_root_path/test/examples/pod_nginx_2.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 25
	if [ $(json_string_value "$($kubectl describe pod $test_pod_1_name)" Phase) != "Ready" ]; then
		return -1
	fi
	if [ $(json_string_value "$($kubectl describe pod $test_pod_2_name)" Phase) != "Ready" ]; then
		return -1
	fi

	$kubectl apply -f $proj_root_path/test/examples/service.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5
	grep -q "$test_pod_1_name" <<<$($kubectl describe service $test_service_name)
	if [ $? -ne 0 ]; then
		return -1
	fi
	grep -q "$test_pod_2_name" <<<$($kubectl describe service $test_service_name)
	if [ $? -ne 0 ]; then
		return -1
	fi
}

test_service_communicate() {
	container_uuid=$(json_string_value "$($kubectl describe pod $test_pod_1_name)" UUID)
	container_name="${container_uuid}_${test_container_name}"
	cluster_ip=$(json_string_value "$($kubectl describe service $test_service_name)" ClusterIP)
	docker exec $container_name /bin/bash -c "curl $cluster_ip:$test_service_port --connect-timeout 5" &>/dev/null
	return $?
}

test_service_update() {
	$kubectl apply -f $proj_root_path/test/examples/pod_nginx_3.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 25
	if [ $(json_string_value "$($kubectl describe pod $test_pod_3_name)" Phase) != "Ready" ]; then
		return -1
	fi

	new_container_uuid=$(json_string_value "$($kubectl describe pod $test_pod_3_name)" UUID)
	new_container_name="${new_container_uuid}_${test_container_name}"
	docker exec $new_container_name /bin/bash -c "curl $cluster_ip:$test_service_port --connect-timeout 5" &>/dev/null
	return $?
}

test_delete_service() {
	$kubectl delete service $test_service_name >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 3
	grep -q "services are not found" <<<$($kubectl describe service $test_service_name)
	return $?
}

test_service_block() {
	docker exec $container_name /bin/bash -c "curl $cluster_ip:$test_service_port --connect-timeout 5" &>/dev/null
	if [ $? -eq 0 ]; then
		return -1
	fi
}

test_create_service
check_test $? "test create service"

test_service_communicate
check_test $? "test service communicate"

test_service_update
check_test $? "test service update"

test_delete_service
check_test $? "test delete service"

test_service_block
check_test $? "test service block"

clean_test_env
