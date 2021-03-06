#!/bin/bash

nginx_container_name=kuberboat-nginx
nginx_config_dir=$HOME/.kube/nginx
coredns_container_name=kuberboat-coredns
coredns_config_dir=$HOME/.kube/coredns

# Argument: container name.
get_container_ip() {
	ip_regex="\"IPAddress\": \"([^\"]*)\""
	[[ $(docker inspect $1) =~ $ip_regex ]]
	if [ ! -z "${BASH_REMATCH[1]}" ]; then
		ip=${BASH_REMATCH[1]}
	else
		echo "cannot find $1 ip address"
		exit -1
		ip=""
	fi
}

# Argument: (component name, container ip)
write_component_ip() {
	docker exec etcd /bin/sh -c "usr/local/bin/etcdctl put /ip/$1 $2" &>/dev/null
	if [ $? -ne 0 ]; then
		echo "cannot set $1 ip to etcd"
		exit -1
	fi
}

# Get master IP.
apiserver_ip=${KUBE_SERVER_IP}
if [ -z $apiserver_ip ]; then
	echo "env variable KUBE_SERVER_IP not set"
	exit -1
fi

# Get etcd container IP address.
get_container_ip "etcd"
etcd_ip=$ip

# Recreate nginx config directory.
rm -rf $nginx_config_dir && mkdir -p $nginx_config_dir &&
	# Start nginx container.
	docker start $nginx_container_name &>/dev/null
if [ $? -ne 0 ]; then
	docker run -d \
		--name $nginx_container_name \
		--restart always \
		-v $nginx_config_dir:/etc/nginx/conf.d \
		-p 80:80 \
		nginx:1.21.6 &>/dev/null
	if [ $? -eq 0 ]; then
		echo "nginx container started, name is ${nginx_container_name}"
	else
		echo "Fail to start nginx"
		exit -1
	fi
else
	echo "nginx container already started"
fi

# Write nginx IP to etcd.
write_component_ip "nginx" $apiserver_ip

# Recreate CoreDNS config directory.
rm -rf $coredns_config_dir && mkdir -p $coredns_config_dir
# Replace the placeholder in Corefile.template with the actual etcd IP, and write it to config directory.
template=$(cat ./assets/Corefile.template)
echo "${template/xxxxxx/$etcd_ip}" >$coredns_config_dir/Corefile
# Disable systemd-resolved (when not on ci).
if [[ $KUBE_CI_MODE != "ON" ]]; then
	systemctl stop systemd-resolved &>/dev/null
	systemctl disable systemd-resolved &>/dev/null
	cat <<EOF
systemd-resolved has been disabled on this machine to free up udp 53 port. 
If you would like to turn systemd-resolved back on (they can be automatically
restarted with stop_dns.sh), please shutdown CoreDNS nameserver, then type
	systemctl enable systemd-resolved
	systemctl start systemd-resolved
EOF
fi
# Start CoreDNS container.
docker start $coredns_container_name &>/dev/null
if [ $? -ne 0 ]; then
	if [[ $KUBE_CI_MODE != "ON" ]]; then
		docker run -d \
			--name $coredns_container_name \
			--restart always \
			-v $coredns_config_dir:/etc/coredns \
			-p 53:53/udp \
			coredns/coredns:1.9.1 \
			-conf /etc/coredns/Corefile &>/dev/null
		if [ $? -eq 0 ]; then
			echo "CoreDNS container started, name is ${coredns_container_name}"
		else
			echo "Fail to start CoreDNS"
			exit -1
		fi
	else
		docker run -d \
			--name $coredns_container_name \
			--restart always \
			-v $coredns_config_dir:/etc/coredns \
			coredns/coredns:1.9.1 \
			-conf /etc/coredns/Corefile &>/dev/null
		if [ $? -eq 0 ]; then
			echo "CoreDNS container started, name is ${coredns_container_name}"
		else
			echo "Fail to start CoreDNS"
			exit -1
		fi
	fi
else
	echo "CoreDNS container already started"
fi

# Write CoreDNS IP to etcd.
get_container_ip $coredns_container_name
write_component_ip "coredns" $ip

# Write CoreDNS's node IP to etcd, so that host machines can also
# access name server by modifying /etc/resolv.conf.
write_component_ip "coredns-host" $apiserver_ip
