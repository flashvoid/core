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

package schema

import (
	"fmt"
	"strings"

	"github.com/romana/core/common/client"
	"github.com/romana/core/labels/types"
)

// NetworkKey makes an etcd key for Network object.
func NetworkKey(network client.Network) string {
	return fmt.Sprintf(Map["Network"], network.Name)
}

// BlockKey makes an etcd key for Block object.
func BlockKey(network client.Network, block client.Block) string {
	return fmt.Sprintf(Map["Block"], network.Name, block.CIDR.String())
}

// EndpointKey make an etcd key for Endpoint object.
func EndpointKey(endpoint types.Endpoint) string {
	cidr := strings.Replace(endpoint.Block, "/", "s", -1)
	return fmt.Sprintf(Map["Endpoint"], endpoint.Network, cidr, endpoint.Name)
}

var Map = map[string]string{
	"Network":  "/obj/networks/%s",
	"Block":    "/obj/networks/%s/blocks/%s",
	"Endpoint": "/obj/networks/%s/blocks/%s/endpoints/%s",
}
