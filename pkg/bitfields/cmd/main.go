package main

import (
	"fmt"
//	"github.com/romana/core/common"
	"github.com/romana/core/pkg/bitfields"
	"github.com/romana/core/common/store"
//	"github.com/romana/core/topology"
)

/*
      store: 
        type: etcd
        host: localhost
        port: 2379
        database: /romana
*/

// Main entry point for the topology microservice
func main() {
	storeConfig := make(map[string]interface{})
	storeConfig["type"] = "etcd"
	storeConfig["host"] = "localhost"
	storeConfig["port"] = 2379
	storeConfig["database"] = "/romana"

	store, err := store.GetStore(storeConfig)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	store.Connect()

	bf, err := bitfields.New(store, "asfix", uint(0), uint(8))
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(bf)
}
