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
