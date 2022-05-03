package etcd

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"p9t.io/kuberboat/pkg/api/core"
)

const REQUEST_TIMEOUT = 2 * time.Second
const DIAL_TIMEOUT = 2 * time.Second

var client *clientv3.Client

func InitializeClient(etcdServers string) error {
	servers := strings.Split(etcdServers, ",")
	// FIXME(WindowsXp): we need to call `cli.Close()` when apiserver is closed, maybe we need a destructor for apiserver
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   servers,
		DialTimeout: DIAL_TIMEOUT,
	})
	if err != nil {
		return err
	}
	client = cli
	return nil
}

func GetPodNames(pods *list.List) []string {
	podNames := make([]string, 0, pods.Len())
	for e := pods.Front(); e != pods.Back(); e = e.Next() {
		podNames = append(podNames, e.Value.(*core.Pod).Name)
	}
	return podNames
}

// Put is a wrapper of clientv3.Put
// Pass the instance of the object you want to store in etcd
func Put(key string, val interface{}, opts ...clientv3.OpOption) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("error marshalling data in etcd: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	_, err = client.Put(ctx, key, string(data), opts...)
	cancel()
	return err
}

// Get is a wrapper of clientv3.Get
// Pass an instance of the type you want to get from etcd and it will return a slice of that type
func Get(key string, valueType interface{}, opts ...clientv3.OpOption) ([]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	resp, err := client.Get(ctx, key, opts...)
	cancel()
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, 0, resp.Count)
	for _, kv := range resp.Kvs {
		buffer := valueType
		if err = json.Unmarshal(kv.Value, &buffer); err != nil {
			return nil, fmt.Errorf("error unmarshalling data in etcd: %v", err)
		}
		values = append(values, buffer)
	}
	return values, nil
}

func GetRaw(key string, opts ...clientv3.OpOption) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	resp, err := client.Get(ctx, key, opts...)
	cancel()
	if err != nil {
		return nil, err
	}
	for _, kv := range resp.Kvs {
		if string(kv.Key) == key {
			return kv.Value, nil
		}
	}
	return nil, fmt.Errorf("key not found: %v", key)
}

// Delete is a wrapper of clientv3.Delete
// Pass the key and the options like WithPrefix etc.
func Delete(key string, opts ...clientv3.OpOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	_, err := client.Delete(ctx, key, opts...)
	cancel()
	return err
}
