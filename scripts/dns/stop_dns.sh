#!/bin/bash

nginx_container_name=kuberboat-nginx
coredns_container_name=kuberboat-coredns

# Stop nginx container.
docker stop $nginx_container_name && docker rm $nginx_container_name

if [ $? -eq 0 ]; then
	echo "successfully stopped nginx container"
else
	echo "fail to stop nginx, please remove it manually"
fi

# Stop coredns contaienr.
docker stop $coredns_container_name && docker rm $coredns_container_name

if [ $? -eq 0 ]; then
	echo "successfully stopped coredns container"
else
	echo "fail to stop coredns, please remove it manually"
fi

# Re-enable systemd-resolved (when not on ci).
if [[ $KUBE_CI_MODE != "ON" ]]; then
	systemctl enable systemd-resolved &>/dev/null
	systemctl start systemd-resolved &>/dev/null
	echo "systemd-resolved re-enabled"
fi
