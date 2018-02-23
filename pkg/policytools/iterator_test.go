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
	"testing"

	"github.com/romana/core/common/api"
)

func TestNewPolicyIterator(t *testing.T) {

	// policyToList := func(p ...api.Policy) []api.Policy { return p }

	// expectFunc is used in test cases to test returns of NewPolicyIterator().
	type expectFunc func(p *PolicyIterator, e error) error

	// expectFunc that asserts an error.
	mustErr := func(p *PolicyIterator, e error) error {
		if e == nil {
			return fmt.Errorf("expected an error, got nothing")
		}

		if p != nil {
			return fmt.Errorf("non nil iterator with error")
		}
		return nil
	}

	// returns expectFunc that counts iterations of iterator
	// and matches them against expected value.
	countIterations := func(i int) expectFunc {
		return func(p *PolicyIterator, e error) error {
			if e != nil {
				return e
			}

			var j int
			for p.Next() {
				j++
				policy, target, peer, rule, direction := p.Items()
				t.Logf("Iterating over %v %v %v %v %v", policy.ID, target, peer, rule, direction)
			}
			if i != j {
				return fmt.Errorf("Unexpected number of iterations, expect %d got %d", i, j)
			}
			return nil
		}
	}

	endpoint1 := api.Endpoint{
		TenantID: "Arthur",
	}

	endpoint2 := api.Endpoint{
		TenantID: "Zaford",
	}

	endpoint3 := api.Endpoint{
		TenantID: "Ford",
	}

	endpoint4 := api.Endpoint{
		TenantID: "Trilian",
	}

	rule1 := api.Rule{
		Protocol: "any",
	}

	rule2 := api.Rule{
		Protocol: "tcp",
		Ports:    []uint{80, 8080},
	}

	ingress1 := api.PolicyBody{
		Peers: []api.Endpoint{endpoint1},
		Rules: []api.Rule{rule1},
	}

	ingress2 := api.PolicyBody{
		Peers: []api.Endpoint{endpoint2},
		Rules: []api.Rule{rule2},
	}

	ingress3 := api.PolicyBody{
		Peers: []api.Endpoint{endpoint3, endpoint4},
		Rules: []api.Rule{rule1, rule2},
	}

	testCases := []struct {
		name     string
		policies []api.Policy
		expect   expectFunc
	}{
		{
			name: "test empty policy",
			policies: []api.Policy{
				api.Policy{
					Direction: api.PolicyDirectionIngress,
					ID:        "empty policy",
				},
			},
			expect: mustErr,
		},
		{
			name: "test policy with empty target",
			policies: []api.Policy{
				api.Policy{
					ID:        "empty policy target",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{},
					Ingress: []api.PolicyBody{
						ingress1,
					},
				},
			},
			expect: mustErr,
		},
		{
			name: "test policy with empty ingress",
			policies: []api.Policy{
				api.Policy{
					ID:        "empty policy ingress",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint1},
					Ingress:   []api.PolicyBody{},
				},
			},
			expect: mustErr,
		},
		{
			name: "test policy with empty peers",
			policies: []api.Policy{
				api.Policy{
					ID:        "empty policy peers",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint1},
					Ingress: []api.PolicyBody{
						api.PolicyBody{
							Rules: []api.Rule{rule1},
						},
					},
				},
			},
			expect: mustErr,
		},
		{
			name: "test policy with empty rules",
			policies: []api.Policy{
				api.Policy{
					ID:        "empty policy rules",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint1},
					Ingress: []api.PolicyBody{
						api.PolicyBody{
							Peers: []api.Endpoint{endpoint1},
						},
					},
				},
			},
			expect: mustErr,
		},
		{
			name:     "test policy list",
			policies: []api.Policy{},
			expect:   mustErr,
		},
		{
			name:     "test nil policy list",
			policies: nil,
			expect:   mustErr,
		},
		{
			name: "test policy with 1 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint2},
					Ingress: []api.PolicyBody{
						ingress1,
					},
				},
			},
			expect: countIterations(1),
		},
		{
			name: "test policy with 4 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint3, endpoint4},
					Ingress: []api.PolicyBody{
						ingress1,
						ingress2,
					},
				},
			},
			expect: countIterations(4),
		},
		{
			name: "test policy with 12 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint3, endpoint4},
					Ingress: []api.PolicyBody{
						ingress1,
						ingress2,
					},
				},
				api.Policy{
					ID:        "policy2",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint1, endpoint2},
					Ingress: []api.PolicyBody{
						ingress3,
					},
				},
			},
			expect: countIterations(12),
		},
		{
			name: "test policy egress with 1 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionEgress,
					AppliedTo: []api.Endpoint{endpoint2},
					Egress: []api.PolicyBody{
						ingress1,
					},
				},
			},
			expect: countIterations(1),
		},
		{
			name: "test policy egress with 4 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionEgress,
					AppliedTo: []api.Endpoint{endpoint3, endpoint4},
					Egress: []api.PolicyBody{
						ingress1,
						ingress2,
					},
				},
			},
			expect: countIterations(4),
		},
		{
			name: "test policy ingress/egress with 12 iterations",
			policies: []api.Policy{
				api.Policy{
					ID:        "policy1",
					Direction: api.PolicyDirectionBoth,
					AppliedTo: []api.Endpoint{endpoint3, endpoint4},
					Ingress: []api.PolicyBody{
						ingress1,
					},
					Egress: []api.PolicyBody{
						ingress2,
					},
				},
				api.Policy{
					ID:        "policy2",
					Direction: api.PolicyDirectionIngress,
					AppliedTo: []api.Endpoint{endpoint1, endpoint2},
					Ingress: []api.PolicyBody{
						ingress3,
					},
				},
			},
			expect: countIterations(12),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			i, err := NewPolicyIterator(testCase.policies)
			err = testCase.expect(i, err)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
