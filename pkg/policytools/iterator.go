// Copyright (c) 2016 Pani Networks
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

package policytools

import (
	"fmt"

	"github.com/romana/core/common/api"
)

// PolicyIterator provides a way to iterate over every combination of a
// target * peer * rule in a list of policies.
type PolicyIterator struct {
	policies  []api.Policy
	policyIdx int
	targetIdx int
	bodyIdx   int
	peerIdx   int
	ruleIdx   int
	started   bool
	direction string
}

// New creates a new PolicyIterator.
func NewPolicyIterator(policies []api.Policy) (*PolicyIterator, error) {
	if len(policies) == 0 {
		return nil, fmt.Errorf("must have non empty policies list")
	}

	emptyBody := func(p api.Policy) bool {
		if p.IsIngress() && p.IsEgress() {
			return len(p.Ingress) == 0 && len(p.Egress) == 0
		}

		if p.IsIngress() {
			return len(p.Ingress) == 0
		}

		if p.IsEgress() {
			return len(p.Egress) == 0
		}

		// should never get here
		return false
	}

	emptyRules := func(p api.Policy) bool {
		for _, i := range p.Ingress {
			if len(i.Rules) == 0 {
				return true
			}
		}
		return false
	}

	emptyPeers := func(p api.Policy) bool {
		for _, i := range p.Ingress {
			if len(i.Peers) == 0 {
				return true
			}
		}
		return false
	}

	emptyTargets := func(p api.Policy) bool {
		return len(p.AppliedTo) == 0
	}

	for _, p := range policies {
		if emptyBody(p) || emptyTargets(p) || emptyPeers(p) || emptyRules(p) {
			return nil, fmt.Errorf("policy %s has .Ingress .AppliedTo .Peers or .Rules field empty", p)
		}
	}

	defaultDirection := api.PolicyDirectionEgress
	if policies[0].IsIngress() {
		defaultDirection = api.PolicyDirectionIngress
	}

	return &PolicyIterator{policies: policies, direction: defaultDirection}, nil
}

// Next advances policy iterator to the next combination
// of policy * target * peer * rule. It returns false if iterator
// can not advance any further, otherwise it returs true.
func (i *PolicyIterator) Next() bool {
	if !i.started {
		i.started = true
		return true
	}

	policy, _, ingress, _, _, _ := i.items()

	if i.ruleIdx < len(ingress.Rules)-1 {
		i.ruleIdx += 1
		return true
	}

	if i.peerIdx < len(ingress.Peers)-1 {
		i.peerIdx += 1
		i.ruleIdx = 0
		return true
	}

	var bodySize int
	switch i.direction {
	case api.PolicyDirectionIngress:
		bodySize = len(policy.Ingress)
	case api.PolicyDirectionEgress:
		bodySize = len(policy.Egress)
	}
	if i.bodyIdx < bodySize-1 {
		i.bodyIdx += 1
		i.ruleIdx = 0
		i.peerIdx = 0
		return true
	}

	if i.targetIdx < len(policy.AppliedTo)-1 {
		i.targetIdx += 1
		i.bodyIdx = 0
		i.ruleIdx = 0
		i.peerIdx = 0
		return true
	}

	if i.direction == api.PolicyDirectionIngress && policy.IsEgress() {
		i.direction = api.PolicyDirectionEgress
		i.bodyIdx = 0
		i.ruleIdx = 0
		i.peerIdx = 0
		i.targetIdx = 0
		return true
	}

	if i.policyIdx < len(i.policies)-1 {
		i.policyIdx += 1
		i.targetIdx = 0
		i.bodyIdx = 0
		i.ruleIdx = 0
		i.peerIdx = 0

		i.direction = api.PolicyDirectionEgress
		if i.policies[i.policyIdx].IsIngress() {
			i.direction = api.PolicyDirectionIngress
		}
		return true
	}

	return false
}

// Items retrieves current combination of policy * target * peer * rule from iterator.
func (i PolicyIterator) Items() (api.Policy, api.Endpoint, api.Endpoint, api.Rule, string) {
	policy, target, _, peer, rule, direction := i.items()
	return policy, target, peer, rule, direction
}

func (i PolicyIterator) items() (api.Policy, api.Endpoint, api.PolicyBody, api.Endpoint, api.Rule, string) {
	policy := i.policies[i.policyIdx]
	target := policy.AppliedTo[i.targetIdx]

	var body api.PolicyBody
	switch i.direction {
	case api.PolicyDirectionIngress:
		body = policy.Ingress[i.bodyIdx]
	case api.PolicyDirectionEgress:
		body = policy.Egress[i.bodyIdx]
	}
	peer := body.Peers[i.peerIdx]
	rule := body.Rules[i.ruleIdx]
	return policy, target, body, peer, rule, i.direction
}
