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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/docker/libkv/store"
	"github.com/romana/core/common"
	"github.com/romana/core/common/client"
	"github.com/romana/core/labels/schema"
	"github.com/romana/core/labels/types"
	log "github.com/romana/rlog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	etcdEndpoints := flag.String("endpoints", "http://192.168.99.10:12379",
		"csv list of etcd endpoints to romana storage")
	etcdPrefix := flag.String("prefix", "/romana",
		"string that prefixes all romana keys in etcd")
	kubeConfigPath := flag.String("kube-config", "/var/run/romana/kubeconfig", "")
	flagFullSync := flag.Int("full-sync", 300, "full sync period")
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

	ipamCh, err := IpamController(ctx, client.DefaultEtcdPrefix+client.IpamDataKey, romanaClient)
	if err != nil {
		log.Errorf("failed to start ipam controller, err=%s", err)
		os.Exit(2)
	}

	kubeClientConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		log.Errorf("failed to make kube config, err=%s", err)
		os.Exit(2)
		// return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Errorf("failed to make kube client, err=%s", err)
		os.Exit(2)
		// return nil, err
	}

	podsChan, podStore, podInformer := PodsController(ctx, kubeClient)
	_ = podInformer

	ticker := time.NewTicker(time.Duration(*flagFullSync) * time.Second)
	TopologySync(ctx, romanaClient, ipamCh, podsChan, podStore, ticker)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	<-c

}

// IpamController produces updates whenever ipam state changes.
func IpamController(ctx context.Context, key string, romanaClient *client.Client) (chan client.IPAM, error) {
	var err error
	ipamCh := make(chan client.IPAM)

	watchChannel, err := romanaClient.Store.WatchExt(key, store.WatcherOptions{}, ctx.Done())

	go func() {
		var ok bool
		var ipamData *store.KVPairExt
		for {
			select {
			case <-ctx.Done():
				return
			case ipamData, ok = <-watchChannel:
				// if channel closed attempt to reconnect
				if !ok {
					log.Errorf("ipam channel closed, ipamData=%v", ipamData)
					watchChannel, err = romanaClient.Store.WatchExt(key,
						store.WatcherOptions{}, ctx.Done())

					log.Errorf("attempting reconnect, err=%v", err)
					time.Sleep(time.Second)
					continue
				}

				// this is unlikely but just in case
				if ipamData == nil {
					log.Errorf("nil from ipam watcher")
					time.Sleep(time.Second)
					continue
				}

				var ipam client.IPAM
				err = json.Unmarshal([]byte(ipamData.Value), &ipam)
				if err != nil {
					log.Errorf("failed to unmarshal ipam err=%s", err)
					time.Sleep(time.Second)
					continue
				}

				ipamCh <- ipam

			}
		}
	}()

	return ipamCh, err
}

// PodEvent represents an event that happened with a pod.
type PodEvent struct {
	Pod   *v1.Pod
	Event string
}

const (
	PodDeleted  = "Deleted"
	PodModified = "Modified"
	PodCreated  = "Created"
)

// PodsController produces events when kubernetes pods are updated.
func PodsController(ctx context.Context, kubeClient *kubernetes.Clientset) (chan PodEvent, cache.Store, *cache.Controller) {
	podsChan := make(chan PodEvent)

	podWatcher := cache.NewListWatchFromClient(
		kubeClient.CoreV1Client.RESTClient(),
		"pods",
		api.NamespaceAll,
		fields.Everything())

	podStore, podInformer := cache.NewInformer(
		podWatcher,
		&v1.Pod{},
		time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if pod, ok := obj.(*v1.Pod); ok {
					podsChan <- PodEvent{pod, PodCreated}
				}
			},
			UpdateFunc: func(old, obj interface{}) {
				if pod, ok := obj.(*v1.Pod); ok {
					podsChan <- PodEvent{pod, PodModified}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if pod, ok := obj.(*v1.Pod); ok {
					podsChan <- PodEvent{pod, PodDeleted}
				}
			},
		},
	)

	go podInformer.Run(ctx.Done())

	return podsChan, podStore, podInformer
}

func FindIpForPod(ipam client.IPAM, pod *v1.Pod) (types.Endpoint, bool) {
	addresPrefix := fmt.Sprintf("%s.%s.", pod.Name, pod.Namespace)

	var findBlockInNetworkDfs func(current *client.Group, ip net.IP) *client.Block
	findBlockInNetworkDfs = func(current *client.Group, ip net.IP) *client.Block {
		// short cut when going down wrong branch
		if !current.CIDR.ContainsIP(ip) && current.Name != "/" {
			log.Tracef(4, "DFS returning from %s since it doesn't contain %s", current.CIDR, ip)
			return nil
		}

		// if not at the leaf yet keep digging
		if len(current.Hosts) == 0 {
			for _, next := range current.Groups {
				block := findBlockInNetworkDfs(next, ip)
				if block != nil {
					log.Tracef(4, "DFS returning from %s with %v", current.CIDR, block)
					return block
				}
			}
		}

		// ok we're at the leaf, lets find our block
		for _, block := range current.Blocks {
			if block.CIDR.ContainsIP(ip) {
				log.Tracef(4, "DFS returning from %s with block %s", current.CIDR, block.CIDR)
				return block
			}
		}

		log.Errorf("should never get here, group=%+v, ip=%+v", current, ip)
		return nil
	}

	findBlockByIP := func(ipam client.IPAM, ip net.IP) (*client.Block, *client.Network, bool) {
		for _, net := range ipam.Networks {
			block := findBlockInNetworkDfs(net.Group, ip)
			if block != nil {
				return block, net, true
			}
		}
		return nil, nil, false
	}

	for k, ip := range ipam.AddressNameToIP {
		if strings.HasPrefix(k, addresPrefix) {
			block, net, ok := findBlockByIP(ipam, ip)
			if ok {
				return types.NewEndpoint(
					pod.Name,
					net.Name,
					block.CIDR.String(),
					ip,
					pod.Labels,
				), true
			}
		}
	}

	return types.Endpoint{}, false
}

func TopologySync(ctx context.Context,
	romanaClient *client.Client,
	ipamCh <-chan client.IPAM,
	podCh <-chan PodEvent,
	podStore cache.Store,
	ticker *time.Ticker,
) {

	var ipam client.IPAM
	var pod PodEvent
	var ok bool

	endpointState := make(map[string]types.Endpoint)

	// this function takes a tree of etcd nodes and
	// extracts all Endpoint objects into an endpointState map
	var dfsNodeWalk func(current *etcd.Node)
	dfsNodeWalk = func(current *etcd.Node) {
		if current.Dir {
			for _, next := range current.Nodes {
				dfsNodeWalk(next)
			}
		}

		var endpoint types.Endpoint
		err := json.Unmarshal([]byte(current.Value), &endpoint)
		if err != nil {
			return
		}

		endpointState[endpoint.Name] = endpoint
		log.Tracef(4, "dfsNodeWalk: %v", endpoint)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Debug("full sync")
				// sync all data from ipam to /romana/obj storage
				// 1) for every pod in kubernetes, find corresponding ip address
				// 2) for every pod+address pair create an Endpoint object under /obj
				// 3) delete all Endpoint objects from /obj if they don't have a
				// corresponding pod

				// fetch etcd.Nodes from /obj key
				var romanaObjectsKey = client.DefaultEtcdPrefix + client.RomanaObjectsPrefix
				res, err := romanaClient.Store.GetExt(romanaObjectsKey,
					store.GetOptions{Recursive: true})
				if err != nil {
					log.Errorf("full sync failed err=%s", err)
				}

				// and extract all Endpoint objects from etcd.Nodes
				endpointState = make(map[string]types.Endpoint)
				dfsNodeWalk(res.Response.Node)

				// assists with deleting outdated entries
				stateUpdated := make(map[string]bool)

				// loop through all the pods
				for _, podIf := range podStore.List() {
					pod, ok := podIf.(*v1.Pod)
					if !ok {
						continue
					}

					// find ip address assigned to pod, if any
					podState, found := FindIpForPod(ipam, pod)
					if !found {
						continue
					}

					// special treatment for pods which are already in
					// endpointState map but their labels have changed
					if lastState, ok := endpointState[podState.Name]; ok {
						// if pod labels are same as stored
						if reflect.DeepEqual(podState.Labels, lastState.Labels) {
							stateUpdated[podState.Name] = true
							continue
						}

						// pod labels updated
						// attempting to store the updated version in database
						err := storeEndpoint(romanaClient, podState)
						if err != nil {
							log.Errorf("failed to store endpoint %v, err=%s",
								podState, err)
							continue
						}
					}

					endpointState[podState.Name] = podState
					stateUpdated[podState.Name] = true
				}

				log.Debugf("endpointState %v", endpointState)
				log.Debugf("stateUpdated %v", stateUpdated)
				// deleting outdated Endpoint objects from database
				var deletedEndpoints []string
				for k, endpoint := range endpointState {
					if _, ok := stateUpdated[endpoint.Name]; !ok {

						log.Debugf("deleting outdated endpoint %v", endpoint)

						err := deleteEndpoint(romanaClient, endpoint)
						if err != nil {
							log.Errorf("failed to delete endpoint %v, err=%s",
								endpoint, err)
							continue
						}

						deletedEndpoints = append(deletedEndpoints, k)
					}
				}

				for _, epName := range deletedEndpoints {
					delete(endpointState, epName)
				}

			case ipam, ok = <-ipamCh:
				// when ipam is updated just update a variable
				// it is going to be used to find ip addresses
				// assigned to the pods
				if ok {
					log.Debugf("ipam update detected %v", ipam)
				}
			case pod, ok = <-podCh:
				// when kubernetes creates or deletes a pod
				// corresponding Endpoint object has to be created/deleted
				// from the database
				if !ok {
					continue
				}

				switch pod.Event {
				case PodCreated, PodModified:
					endpoint, ok := FindIpForPod(ipam, pod.Pod)
					if !ok {
						log.Debugf("Pod %v updated but no corresponding endpoint found in ipam", pod.Pod)
						continue
					}

					err := storeEndpoint(romanaClient, endpoint)
					if err != nil {
						log.Errorf("failed to store endpoint %v, err=%s", endpoint, err)
						continue
					}

				case PodDeleted:
					endpoint, ok := FindIpForPod(ipam, pod.Pod)
					if !ok {
						log.Debugf("Pod %v updated but no corresponding endpoint found in ipam", pod.Pod)
						continue
					}

					err := deleteEndpoint(romanaClient, endpoint)
					if err != nil {
						log.Errorf("failed to delete endpoint %v, err=%s", endpoint, err)
						continue
					}
				}
			}
		}
	}()
}

// storeEndpoint is awrapper that stores endpoint into a database
func storeEndpoint(romanaClient *client.Client, endpoint types.Endpoint) error {
	data, err := json.Marshal(endpoint)
	if err != nil {
		return err
	}

	err = romanaClient.Store.Put(client.DefaultEtcdPrefix+schema.EndpointKey(endpoint), data, nil)
	if err != nil {
		return err
	}

	return nil
}

// deleteEndpoint is a wrapper that deletes endpoint from database
func deleteEndpoint(romanaClient *client.Client, endpoint types.Endpoint) error {
	_, err := romanaClient.Store.Delete(schema.EndpointKey(endpoint))
	return err
}
