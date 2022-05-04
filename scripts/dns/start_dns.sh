#!/bin/bash

nginx_container_name=kuberboat-nginx
nginx_config_dir=$HOME/.kube/nginx
coredns_container_name=kuberboat-coredns
coredns_config_dir=$HOME/.kube/coredns

# Argument: container name.
get_container_ip() {
    ip_regex="\"IPAddress\": \"([^\"]*)\""
    [[ $(docker inspect $1) =~ $ip_regex ]]
    if [ ! -z "${BASH_REMATCH[1]}" ]
    then 
        ip=${BASH_REMATCH[1]}
    else 
        echo "cannot find $1 ip address"; exit -1
        ip=""
    fi
}

# Argument: (component name, container ip)
write_component_ip() {
    docker exec etcd /bin/sh -c "usr/local/bin/etcdctl put /ip/$1 $2" &> /dev/null
    if [ $? -ne 0 ]
    then
        echo "cannot set $1 ip to etcd"; exit -1
    fi
}

# Get etcd container IP address.
get_container_ip "etcd"
etcd_ip=$ip

# Recreate nginx config directory.
rm -rf $nginx_config_dir && mkdir -p $nginx_config_dir && \
# Start nginx container.
docker start $nginx_container_name &> /dev/null
if [ $? -ne 0 ]
then
    docker run -d \
        --name $nginx_container_name \
        --restart always \
        -v $nginx_config_dir:/etc/nginx/conf.d \
        nginx:1.21.6 &> /dev/null && \
    echo "nginx container started, name is ${nginx_container_name}"
else
    echo "nginx container already started"
fi

# Write nginx IP to etcd.
get_container_ip $nginx_container_name
write_component_ip "nginx" $ip

# Recreate CoreDNS config directory.
rm -rf $coredns_config_dir && mkdir -p $coredns_config_dir
# Replace the placeholder in Corefile.template with the actual etcd IP, and write it to config directory.
template=$(cat ./assets/Corefile.template)
echo "${template/xxxxxx/$etcd_ip}" > $coredns_config_dir/Corefile
# Start CoreDNS container.
docker start $coredns_container_name &> /dev/null
if [ $? -ne 0 ]
then
    docker run -d \
        --name $coredns_container_name \
        --restart always \
        -v $coredns_config_dir:/etc/coredns \
        coredns/coredns:1.9.1 \
        -conf /etc/coredns/Corefile &> /dev/null && \
    echo "coredns container started, name is ${coredns_container_name}"
else
    echo "coredns container already started"
fi

# Write CoreDNS IP to etcd.
get_container_ip $coredns_container_name
write_component_ip "coredns" $ip