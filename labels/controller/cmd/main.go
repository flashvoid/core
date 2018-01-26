package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"

	"github.com/romana/core/common"
	"github.com/romana/core/common/client"
	"github.com/romana/core/labels/controller"
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

	store, eChan, err := controller.EndpointController(ctx, romanaClient, "/romana/obj")

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
