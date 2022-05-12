#!/bin/bash

parent_path=$(
	cd "$(dirname "${BASH_SOURCE[0]}")"
	pwd -P
)
source $parent_path/test_util.sh

kubelet_log=$proj_root_path/out/log/kubelet.log

test_register_node() {
	$kubectl apply -f $proj_root_path/test/examples/node.yaml &>/dev/null
	sleep 2
	grep -q "connected to api server" $kubelet_log
	return $?
}

test_register_node
check_test $? "test register node"
