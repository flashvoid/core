package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/docker/libkv/store"
	"github.com/romana/core/agent/cache"
	"github.com/romana/core/common"
	"github.com/romana/core/common/client"
	"github.com/romana/core/labels/types"
	log "github.com/romana/rlog"
)

func main() {
	etcdEndpoints := flag.String("endpoints", "http://192.168.99.10:12379",
		"csv list of etcd endpoints to romana storage")
	etcdPrefix := flag.String("prefix", "/romana",
		"string that prefixes all romana keys in etcd")
	flag.Parse()

	romanaConfig := common.Config{
		EtcdEndpoints: strings.Split(*etcdEndpoints, ","),
		EtcdPrefix:    *etcdPrefix,
	}

	romanaClient, err := client.NewClient(&romanaConfig)
	if err != nil {
		log.Errorf("Failed to initialize romana client: %v", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, eChan, err := EndpointController(ctx, romanaClient, "/romana/obj")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-eChan:
				// log.Infof("received event %v", event)
				log.Infof("current storage keys %v", store.Keys())
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	<-c
}

type Store interface {
	Put(string, types.Endpoint)
	Get(string) (types.Endpoint, bool)
	Delete(string)
	List() []types.Endpoint
	Keys() []string
}

type EndpointStorage struct {
	store cache.Interface
}

func NewEndpointStorage() Store {
	return &EndpointStorage{cache.New()}
}

func (p *EndpointStorage) Put(key string, endpoint types.Endpoint) {
	p.store.Put(key, endpoint)
}

func (p *EndpointStorage) Get(key string) (types.Endpoint, bool) {
	item, ok := p.store.Get(key)
	if !ok {
		return types.Endpoint{}, ok
	}

	endpoint, ok := item.(types.Endpoint)
	if !ok {
		return types.Endpoint{}, ok
	}

	return endpoint, ok
}

func (p *EndpointStorage) List() []types.Endpoint {
	var result []types.Endpoint
	items := p.store.List()
	for _, item := range items {
		endpoint, ok := item.(types.Endpoint)
		if !ok {
			continue
		}
		result = append(result, endpoint)
	}
	return result
}

func (p *EndpointStorage) Keys() []string {
	return p.store.Keys()
}

func (p *EndpointStorage) Delete(key string) {
	p.store.Delete(key)
}

func EndpointController(ctx context.Context, romanaClient *client.Client, key string) (Store, chan types.EndpointEvent, error) {
	var err error
	out := make(chan types.EndpointEvent)
	eStore := NewEndpointStorage()

	watchChannel, err := romanaClient.Store.WatchExt(key, store.WatcherOptions{Recursive: true}, ctx.Done())

	go func() {
		var ok bool
		var endpointData *store.KVPairExt
		for {
			select {
			case <-ctx.Done():
				return
			case endpointData, ok = <-watchChannel:
				// if channel closed attempt to reconnect
				if !ok {
					log.Errorf("endpoint  channel closed, response=%v", endpointData)
					watchChannel, err = romanaClient.Store.WatchExt(key,
						store.WatcherOptions{Recursive: true}, ctx.Done())

					log.Errorf("attempting reconnect, err=%v", err)
					time.Sleep(time.Second)
					continue
				}

				// this is unlikely but just in case
				if endpointData == nil {
					log.Errorf("nil from endpoint watcher")
					time.Sleep(time.Second)
					continue
				}

				log.Infof("etcd event %s for %s", endpointData.Action, endpointData.Key)
				if endpointData.Dir {
					log.Infof("is a directory")
					continue
				}

				var data string
				switch endpointData.Action {
				case "create", "set", "update", "compareAndSwap":
					data = endpointData.Value
				case "delete", "compareAndDelete", "expire":
					data = endpointData.PrevValue
				}

				var endpoint types.Endpoint
				err = json.Unmarshal([]byte(data), &endpoint)
				if err != nil {
					log.Infof("failed to unmarshal endpoint %s, err=%s", data, err)
					continue
				}

				endpointEvent := types.EndpointEvent{
					Endpoint: endpoint,
					Event:    endpointData.Action,
				}

				log.Infof("Received event %v", endpointEvent)

				switch endpointData.Action {
				case "create", "set", "update", "compareAndSwap":
					eStore.Put(endpoint.IP.String(), endpoint)
				case "delete", "compareAndDelete", "expire":
					eStore.Delete(endpoint.IP.String())
				}

				out <- endpointEvent
			}
		}
	}()

	return eStore, out, err
}
