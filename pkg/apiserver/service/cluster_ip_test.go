package service

import (
	"flag"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func TestNextClusterIP(t *testing.T) {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return
	}
	flag.Parse()

	ca, err := NewClusterIPAssigner()
	if err != nil {
		glog.Fatal(err)
	}

	expectedFirstIP := "240.0.0.1"
	firstIP, err := ca.NextClusterIP()
	if err != nil {
		glog.Fatal(err)
	}
	assert.Equal(t, expectedFirstIP, firstIP)

	expectedIP := "240.0.1.0"
	for i := 0; i < 254; i++ {
		_, err = ca.NextClusterIP()
		if err != nil {
			glog.Fatal(err)
		}
	}
	nextIP, err := ca.NextClusterIP()
	if err != nil {
		glog.Fatal(err)
	}
	assert.Equal(t, nextIP, expectedIP)
}
