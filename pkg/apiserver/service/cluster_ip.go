package service

import (
	"fmt"
	"net"
)

const (
	ClusterIPRangeString string = "240.0.0.0/4"
)

type clusterIPAssigner struct {
	clusterIPRange *net.IPNet
	currentIP      net.IP
}

func NewClusterIPAssigner() (*clusterIPAssigner, error) {
	firstIP, clusterIPRange, err := net.ParseCIDR(ClusterIPRangeString)
	if err != nil {
		return nil, err
	}
	return &clusterIPAssigner{
		clusterIPRange: clusterIPRange,
		currentIP:      firstIP,
	}, nil
}

func (ca *clusterIPAssigner) NextClusterIP() (string, error) {
	var newIP net.IP
	octets := ca.currentIP.To4()
	if octets[3]+1 != 0 {
		newIP = net.IPv4(octets[0], octets[1], octets[2], octets[3]+1)
	} else if octets[2]+1 != 0 {
		newIP = net.IPv4(octets[0], octets[1], octets[2]+1, 0)
	} else if octets[1]+1 != 0 {
		newIP = net.IPv4(octets[0], octets[1]+1, 0, 0)
	} else if octets[0]+1 != 0 {
		newIP = net.IPv4(octets[0]+1, 0, 0, 0)
	} else {
		return "", fmt.Errorf("no cluster ip can be assigned")
	}
	if !ca.clusterIPRange.Contains(newIP) {
		return "", fmt.Errorf("no cluster ip can be assigned")
	}
	ca.currentIP = newIP

	// TODO: persist to etcd

	return ca.currentIP.To4().String(), nil
}
