// Copyright (c) 2016-2017 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package controller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/docker/libkv/store"
	"github.com/romana/core/agent/cache"
	"github.com/romana/core/common/client"
	"github.com/romana/core/labels/types"
	log "github.com/romana/rlog"
)

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
