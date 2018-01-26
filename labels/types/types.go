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

package types

import "net"

type EndpointEvent struct {
	Event    string
	Endpoint Endpoint
}

// Endpoint represents IP address and associated attributes.
type Endpoint struct {
	Kind    string
	Name    string
	IP      net.IP
	Labels  map[string]string
	Network string
	Block   string
}

// NewEndpoint initializes new Endpoint.
func NewEndpoint(name, network, block string, ip net.IP, labels map[string]string) Endpoint {
	return Endpoint{
		Kind:    "Endpoint",
		Name:    name,
		IP:      ip,
		Labels:  labels,
		Network: network,
		Block:   block,
	}
}
