package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/docker/libkv/store"
	"github.com/romana/core/common"
	"github.com/romana/core/common/client"
	log "github.com/romana/rlog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ipamKey     = "/ipam"
	ipamDataKey = ipamKey + "/data"
)

func main() {
	etcdEndpoints := flag.String("endpoints", "http://192.168.99.10:12379",
		"csv list of etcd endpoints to romana storage")
	etcdPrefix := flag.String("prefix", "/romana",
		"string that prefixes all romana keys in etcd")
	kubeConfigPath := flag.String("kube-config", "/var/run/romana/kubeconfig", "")
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

	ipamCh, err := IpamController(ctx, *etcdPrefix+ipamDataKey, romanaClient)
	if err != nil {
		log.Errorf("failed to start ipam controller, err=%s", err)
		os.Exit(2)
	}

	/*
		ipam := <-ipamCh

		for k, v := range ipam.AddressNameToIP {
			fmt.Printf("%s = %s\n", k, v)
		}
	*/

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
	/*
		go func() {
			for {
				_ = <-podsChan
			}
		}()

		for i := 0; i < 10; i++ {
			if podInformer.HasSynced() {
				break
			}
			time.Sleep(time.Second)
		}

		if !podInformer.HasSynced() {
			log.Errorf("Pod informer hasn't synced")
			os.Exit(2)
		}

		pods := podStore.List()
		for _, podIf := range pods {
			pod, ok := podIf.(*v1.Pod)
			if !ok {
				log.Errorf("skipping pod %v because it's no pod", podIf)
				continue
			}
			podName := fmt.Sprintf("%s.%s.", pod.Name, pod.Namespace)
			fmt.Printf("%s\n", podName)
			FindIpForPod(ipam, pod)
		}
	*/

	TopologySync(ctx, romanaClient, ipamCh, podsChan, podStore)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	<-c

}

func IpamController(ctx context.Context, key string, romanaClient *client.Client) (chan client.IPAM, error) {
	var err error
	ipamCh := make(chan client.IPAM)

	watchChannel, err := romanaClient.Store.WatchExt(ipamDataKey, store.WatcherOptions{}, ctx.Done())

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

type PodEvent struct {
	Pod   *v1.Pod
	Event string
}

const (
	PodDeleted  = "Deleted"
	PodModified = "Modified"
	PodCreated  = "Created"
)

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

type Endpoint struct {
	Kind    string
	Name    string
	Ip      net.IP
	Labels  map[string]string
	Network string
	Block   string
}

func NewEndpoint(name, network, block string, ip net.IP, labels map[string]string) Endpoint {
	return Endpoint{
		Kind:    "Endpoint",
		Name:    name,
		Ip:      ip,
		Labels:  labels,
		Network: network,
		Block:   block,
	}
}

func FindIpForPod(ipam client.IPAM, pod *v1.Pod) (Endpoint, bool) {
	addresPrefix := fmt.Sprintf("%s.%s.", pod.Name, pod.Namespace)

	var findBlockInNetworkDfs func(current *client.Group, ip net.IP) *client.Block
	findBlockInNetworkDfs = func(current *client.Group, ip net.IP) *client.Block {
		// short cut when going down wrong branch
		if !current.CIDR.ContainsIP(ip) && current.Name != "/" {
			log.Infof("DFS returning from %s since it doesn't contain %s", current.CIDR, ip)
			return nil
		}

		// if not at the leaf yet keep digging
		if len(current.Hosts) == 0 {
			for _, next := range current.Groups {
				block := findBlockInNetworkDfs(next, ip)
				if block != nil {
					log.Infof("DFS returning from %s with %v", current.CIDR, block)
					return block
				}
			}
		}

		// ok we're at the leaf, lets find our block
		for _, block := range current.Blocks {
			if block.CIDR.ContainsIP(ip) {
				log.Infof("DFS returning from %s with block %s", current.CIDR, block.CIDR)
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
				return NewEndpoint(
					pod.Name,
					net.Name,
					block.CIDR.String(),
					ip,
					pod.Labels,
				), true
			}
		}
	}

	return Endpoint{}, false
}

func TopologySync(ctx context.Context,
	romanaClient *client.Client,
	ipamCh <-chan client.IPAM,
	podCh <-chan PodEvent,
	podStore cache.Store,
) {

	var ipam client.IPAM
	var pod PodEvent
	var ok bool

	ticker := time.NewTicker(time.Duration(time.Second * 5))

	endpointState := make(map[string]Endpoint)

	var dfsNodeWalk func(current *etcd.Node)
	dfsNodeWalk = func(current *etcd.Node) {
		if current.Dir {
			for _, next := range current.Nodes {
				dfsNodeWalk(next)
			}
		}

		var endpoint Endpoint
		err := json.Unmarshal([]byte(current.Value), &endpoint)
		if err != nil {
			return
		}

		endpointState[endpoint.Name] = endpoint
		log.Infof("dfsNodeWalk: %v", endpoint)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res, err := romanaClient.Store.GetExt("/romana/obj",
					store.GetOptions{Recursive: true})
				if err != nil {
					log.Errorf("full sync failed err=%s", err)
				}
				dfsNodeWalk(res.Response.Node)
			case ipam, ok = <-ipamCh:
				if ok {
					fmt.Println(ipam)
				}
			case pod, ok = <-podCh:
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

					data, err := json.Marshal(endpoint)
					if err != nil {
						log.Errorf("failed to marshal endpoint %+v", endpoint)
						continue
					}

					err = romanaClient.Store.Put(schema.EndpointKey(endpoint), data, nil)
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

					ok, err := romanaClient.Store.Delete(schema.EndpointKey(endpoint))
					if err != nil || !ok {
						log.Errorf("failed to delete endpoint %v, err=%s", endpoint, err)
						continue
					}
				}
			}
		}
	}()
}

type schematic struct {
	Prefix string
	Map    map[string]string
}

func (s schematic) NetworkKey(network client.Network) string {
	return fmt.Sprintf(s.Map["Network"], s.Prefix, network.Name)
}

func (s schematic) BlockKey(network client.Network, block client.Block) string {
	return fmt.Sprintf(s.Map["Block"], s.Prefix, network.Name, block.CIDR.String())
}

func (s schematic) EndpointKey(endpoint Endpoint) string {
	cidr := strings.Replace(endpoint.Block, "/", "s", -1)
	return fmt.Sprintf(s.Map["Endpoint"], s.Prefix, endpoint.Network, cidr, endpoint.Name)
}

var schema = schematic{
	Prefix: "/romana",
	Map: map[string]string{
		"Network":  "%s/obj/networks/%s",
		"Block":    "%s/obj/networks/%s/blocks/%s",
		"Endpoint": "%s/obj/networks/%s/blocks/%s/endpoints/%s",
	},
}
