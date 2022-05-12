#!/bin/bash

if [ -z $(pgrep etcd) ]; then
	echo "etcd not started"
	exit 0
fi

api_objects=("/Pods" "/Deployments" "/Services" "/Nodes")

echo "clearing etcd"
etcdctl version &>/dev/null

if [ $? -ne 0 ]; then
	echo "cannot find etcdctl locally, try docker exec"
	for i in "${api_objects[@]}"; do
		docker exec etcd /bin/sh -c "/usr/local/bin/etcdctl del $i --prefix"
	done
else
	for i in "${api_objects[@]}"; do
		etcdctl del $i --prefix
	done
fi
