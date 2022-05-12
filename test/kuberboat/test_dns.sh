#!/bin/bash

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

# Test object info. Should be consistent with YAML file.
test_deplyment_name="deployment-nginx" # deployment_nginx.yaml
test_image="nginx:1.21.6"              # deployment_nginx.yaml
test_dns_name="dns-example"            # dns.yaml
test_url="test.com/aaa"                # dns.yaml

test_create_dns() {
	$kubectl apply -f $proj_root_path/test/examples/deployment_nginx.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 10
	$kubectl apply -f $proj_root_path/test/examples/service.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5
	$kubectl apply -f $proj_root_path/test/examples/dns.yaml >/dev/null
	if [ $? -ne 0 ]; then
		return -1
	fi
	sleep 5

	# The nginx containers should be the first and the third one.
	first_container_id=$(docker container ls -q --filter name=$test_deployment_name --filter ancestor=$test_image | head -n 1)
	second_container_id=$(docker container ls -q --filter name=$test_deployment_name --filter ancestor=$test_image | head -n 2 | tail -1)

	if [ -z first_container_id ] || [ -z second_container_id ]; then
		return -1
	fi
	output=$(docker exec $first_container_id /bin/bash -c "curl $test_url --connect-timeout 5" 2>&1)
	if ! echo $output | grep -q "$nginx_welcome"; then
		return -1
	fi
	output=$(docker exec $second_container_id /bin/bash -c "curl $test_url --connect-timeout 5" 2>&1)
	if ! echo $output | grep -q "$nginx_welcome"; then
		return -1
	fi
	$kubectl delete services --all >/dev/null
	$kubectl delete deployments --all >/dev/null
	return 0
}

test_create_dns
check_test $? "test create dns"
