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

// Policy enforcer package translates romana policies into iptables rules.
package enforcer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	utilexec "github.com/romana/core/agent/exec"
	"github.com/romana/core/agent/iptsave"
	"github.com/romana/core/agent/policycache"
	"github.com/romana/core/common/api"
	"github.com/romana/core/common/log/trace"
	"github.com/romana/core/labels/controller"
	"github.com/romana/core/labels/expression"
	"github.com/romana/core/labels/types"
	"github.com/romana/core/pkg/policytools"

	"github.com/romana/ipset"
	log "github.com/romana/rlog"
)

// Interface defines policy enforcer behavior.
type Interface interface {
	// Run starts internal loop that handles updates from policies.
	Run(context.Context)
}

// Endpoint implements Interface.
type Enforcer struct {
	endpointStore controller.Store

	endpointChan <-chan types.EndpointEvent

	endpointUpdate bool

	// provides access to in memeory policy cache.
	policyCache policycache.Interface

	// provides updates about romana policies.
	policies <-chan api.Policy

	// updates about romana blocksChannel
	blocksChannel <-chan api.IPAMBlocksResponse

	// blocks
	blocks api.IPAMBlocksResponse

	// name of a current host.
	hostname string

	// blocksUpdate holds hash associated with last update of tenant cache.
	blocksUpdate bool

	// policyUpdate holds hash associated with last update of policy cache.
	policyUpdate bool

	// Delay between main loop runs.
	ticker *time.Ticker

	// exec used to apply iptables policies.
	exec utilexec.Executable

	// attempt to refresh policies every refreshSeconds.
	refreshSeconds int
}

// New returns new policy enforcer.
func New(endpointStore controller.Store,
	endpointChan <-chan types.EndpointEvent,
	policy policycache.Interface,
	policies <-chan api.Policy,
	blocks api.IPAMBlocksResponse,
	blocksChannel <-chan api.IPAMBlocksResponse,
	hostname string,
	utilexec utilexec.Executable,
	refreshSeconds int) (Interface, error) {

	var err error

	if IptablesSaveBin, err = exec.LookPath("iptables-save"); err != nil {
		return nil, err
	}

	if IptablesRestoreBin, err = exec.LookPath("iptables-restore"); err != nil {
		return nil, err
	}

	return &Enforcer{
		endpointStore:  endpointStore,
		endpointChan:   endpointChan,
		policyCache:    policy,
		policies:       policies,
		blocks:         blocks,
		blocksChannel:  blocksChannel,
		hostname:       hostname,
		exec:           utilexec,
		refreshSeconds: refreshSeconds,
	}, nil
}

// Run implements Interface.  It reads notifications
// from the policy cache and from the block cache,
// when either cache chagned re-renders all iptables rules.
func (a *Enforcer) Run(ctx context.Context) {
	log.Trace(trace.Public, "Policy enforcer Run()")

	var romanaBlocks []api.IPAMBlockResponse
	romanaBlocks = a.blocks.Blocks

	iptables := &iptsave.IPtables{}
	a.ticker = time.NewTicker(time.Duration(a.refreshSeconds) * time.Second)

	go func() {
		for {
			select {
			case <-a.ticker.C:
				if !a.policyUpdate && !a.blocksUpdate && !a.endpointUpdate {
					log.Tracef(5, "Policy enforcer tick skipped due no updates")
					continue
				}

				if len(romanaBlocks) == 0 {
					log.Trace(5, "no blocks, skipping")
					continue
				}
				NumEnforcerTick.Inc()

				sets, err := makeBlockSets(romanaBlocks, a.policyCache, a.endpointStore, a.hostname)
				if err != nil {
					log.Errorf("Failed to update ipsets, can't apply Romana policies, %s", err)
					ErrMakeSets.Inc()
					continue
				}

				err = updateIpsets(ctx, sets)
				if err != nil {
					log.Errorf("Failed to update ipsets, can't apply Romana policies, %s", err)
					ErrApplySets.Inc()
					continue
				}
				NumBlockUpdates.Inc()
				NumManagedSets.Set(float64(len(sets.Sets)))

				iptables = renderIPtables(a.policyCache, a.hostname, romanaBlocks)
				cleanupUnusedChains(iptables, a.exec)
				if ValidateIPtables(iptables, a.exec) {
					if err := ApplyIPtables(iptables, a.exec); err != nil {
						log.Errorf("iptables-restore call failed %s", err)
						ErrApplyIptables.Inc()
					}
					log.Tracef(6, "Applied iptables rules\n%s", iptables.Render())

				} else {
					ErrValidateIptables.Inc()
					log.Tracef(6, "Failed to validate iptables\n%s%n", iptables.Render())
				}
				NumPolicyUpdates.Inc()

				a.policyUpdate = false
				a.blocksUpdate = false
				a.endpointUpdate = false

			case blocksList := <-a.blocksChannel:
				log.Trace(4, "Policy enforcer receives update from cache blocks revision=%d",
					blocksList.Revision)
				romanaBlocks = blocksList.Blocks
				a.blocksUpdate = true

			case <-a.policies:
				log.Trace(4, "Policy enforcer receives update from policy cache")
				a.policyUpdate = true

			case <-a.endpointChan:
				log.Trace(4, "Endpoints update detected")
				a.endpointUpdate = true

			case <-ctx.Done():
				log.Infof("Policy enforcer stopping")
				a.ticker.Stop()
				return
			}
		}
	}()
}

// makeBlockSets creates ipset configuration for policies and blocks.
func makeBlockSets(blocks []api.IPAMBlockResponse, policyCache policycache.Interface, endpointStore controller.Store, hostname string) (*ipset.Ipset, error) {
	policies := policyCache.List()
	sets := ipset.NewIpset()

	// for every policy produce a set to match policy related traffic.
	for _, policy := range policies {
		policySetTarget, policySetPeer, err := makePolicySets(policy, endpointStore)
		if err != nil {
			return nil, err
		}

		err = sets.AddSet(policySetTarget)
		if err != nil {
			return nil, err
		}

		err = sets.AddSet(policySetPeer)
		if err != nil {
			return nil, err
		}
	}

	// for every block produce 2 sets
	// - tenant+segment set contains all the blocks
	// for the relevan t+s combination
	// - tenant set contains all the t+s sets for the
	// relevant tenant
	for _, block := range blocks {
		/*
			if block.Segment == "" {
				// TODO error, can't distinguish between
				// tenantSegment set and tenant set
			}
		*/

		// TODO ignore blocks for other hostnames? then what about egress?
		log.Tracef(5, "Making set for %+v", block)

		segmentSetName := policytools.MakeTenantSetName(block.Tenant, block.Segment)
		segmentSet := sets.SetByName(segmentSetName)
		if segmentSet == nil {
			segmentSet, _ = ipset.NewSet(segmentSetName, ipset.SetHashNet)
		}
		err := ipset.SuppressItemExist(sets.AddSet(segmentSet))
		if err != nil {
			return nil, err
		}

		memberForSegmentSet, _ := ipset.NewMember(block.CIDR.IPNet.String(), segmentSet)
		err = ipset.SuppressItemExist(segmentSet.AddMember(memberForSegmentSet))
		if err != nil {
			return nil, err
		}

		tenantSetName := policytools.MakeTenantSetName(block.Tenant, "")
		tenantSet := sets.SetByName(tenantSetName)
		if tenantSet == nil {
			tenantSet, _ = ipset.NewSet(tenantSetName, ipset.SetListSet)
		}
		err = ipset.SuppressItemExist(sets.AddSet(tenantSet))
		if err != nil {
			return nil, err
		}

		memberForTenantSet, _ := ipset.NewMember(segmentSet.Name, tenantSet)
		err = ipset.SuppressItemExist(tenantSet.AddMember(memberForTenantSet))
		if err != nil {
			return nil, err
		}

	}

	// makes one set that has all the blocks for current host
	localBlocksSet, err := ipset.NewSet(LocalBlockSetName, ipset.SetHashNet)
	if err != nil {
		return nil, err
	}
	for _, block := range blocks {
		if block.Host == hostname {
			localMemeber, _ := ipset.NewMember(block.CIDR.String(), localBlocksSet)
			err := ipset.SuppressItemExist(localBlocksSet.AddMember(localMemeber))
			if err != nil {
				return nil, err
			}
		}
	}
	err = ipset.SuppressItemExist(sets.AddSet(localBlocksSet))
	if err != nil {
		return nil, err
	}

	return sets, nil
}

// LocalBlockSetName is an ipset set that matches traffic for endpoints
// located on current host.
const LocalBlockSetName = "localBlocks"

// makePolicySets produces a set that matches traffic selected by policy Peer/AppliedTo fields.
func makePolicySets(policy api.Policy, endpointStore controller.Store) (*ipset.Set, *ipset.Set, error) {
	var policySetPeer, policySetTarget *ipset.Set
	var err error

	policySetTarget, err = ipset.NewSet(
		policytools.MakeRomanaPolicyNameIpsetTarget(policy), ipset.SetHashNet)
	if err != nil {
		return nil, nil, err
	}

	policySetPeer, err = ipset.NewSet(
		policytools.MakeRomanaPolicyNameIpsetPeer(policy), ipset.SetHashNet)
	if err != nil {
		return nil, nil, err
	}

	// resolves any labels, found in endpoints list, to the IP addresses
	// of known pods and adds discovered IP addresses to the set.
	fillPolicySet := func(endpoints []api.Endpoint, set *ipset.Set) error {
		for _, endpoint := range endpoints {

			// If endpoint has labels provided - find endpoints with corresponding
			// labels in endpoint store and add these endpoint IPs to the set.
			if len(endpoint.Labels) > 0 {
				labelExpression := expression.ExpressionFromMap(endpoint.Labels)

				for _, endpoint := range endpointStore.List() {
					if labelExpression.Eval(endpoint.Labels) {
						member, err := ipset.NewMember(
							fmt.Sprintf("%s/32", endpoint.IP),
							set,
						)
						if err != nil {
							return err
						}

						err = ipset.SuppressItemExist(set.AddMember(member))
						if err != nil {
							return err
						}
					}
				}
			}

			// If endpoint has CIDR field, add this cidr to the policy set.
			peerType := policytools.DetectPolicyPeerType(endpoint)
			if peerType != policytools.PeerCIDR {
				continue
			}

			member, err := ipset.NewMember(endpoint.Cidr, set)
			if err != nil {
				return err
			}

			err = ipset.SuppressItemExist(set.AddMember(member))
			if err != nil {
				return err
			}
		}

		return nil
	}

	if policy.IsIngress() {
		for _, ingress := range policy.Ingress {
			err := fillPolicySet(ingress.Peers, policySetPeer)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	if policy.IsEgress() {
		for _, egress := range policy.Egress {
			err := fillPolicySet(egress.Peers, policySetTarget)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	// create ipset records to match policy target by labels
	for _, target := range policy.AppliedTo {
		// if no labels defined then skip target
		if len(target.Labels) == 0 {
			continue
		}

		labelExpression := expression.ExpressionFromMap(target.Labels)
		for _, endpoint := range endpointStore.List() {
			if labelExpression.Eval(endpoint.Labels) {
				member, err := ipset.NewMember(
					fmt.Sprintf("%s/32", endpoint.IP),
					policySetTarget,
				)
				if err != nil {
					return nil, nil, err
				}

				err = ipset.SuppressItemExist(policySetTarget.AddMember(member))
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	return policySetTarget, policySetPeer, nil
}

// validateFunc is a signature for a function that validates api.Endpoint
// according to some criteria.
type validateFunc func(target api.Endpoint) bool

// renderIPtables creates iptables rules for all romana policies in policy cache
// except the ones which depends on non-existend tenant/segment.
func renderIPtables(policyCache policycache.Interface, hostname string, blocks []api.IPAMBlockResponse) *iptsave.IPtables {
	log.Trace(trace.Private, "Policy enforcer in renderIPtables()")

	// Make empty iptables object.
	iptables := iptsave.IPtables{
		Tables: []*iptsave.IPtable{
			&iptsave.IPtable{
				Name: "filter",
			},
		},
	}

	// filter out blocks that are assigned to remote hosts,
	// this should prevent policies being created
	// across an entire cluster.
	var localBlocks []api.IPAMBlockResponse
	for _, block := range blocks {
		if block.Host == hostname {
			localBlocks = append(localBlocks, block)
		}
	}

	// validateTargetForHost returns validateFunc that only accepts
	// targets which have endpoints on current host.
	validateTargetForHost := func(blocks []api.IPAMBlockResponse) validateFunc {
		return func(target api.Endpoint) bool {
			return targetValid(target, blocks)
		}
	}

	makeBase(&iptables)
	makePolicies(policyCache.List(), validateTargetForHost(localBlocks), &iptables)

	return &iptables
}

// makeBase populates iptables with romana chains that do not depend on presence
// if any external resource like tenant and policy chains do.
func makeBase(iptables *iptsave.IPtables) {
	// For now our policies only exist in a filter tables so we don't care
	// for other tables.
	filter := iptables.TableByName("filter")
	filter.Chains = MakeBaseRules()

}

// makePolicies populates policy related rules into the iptables.
func makePolicies(policies []api.Policy, valid validateFunc, iptables *iptsave.IPtables) {
	log.Trace(trace.Private, "Policy enforcer in makePolicies()")

	// iterator iterates over each combination of
	// policy * target * peer * rule.
	iterator, err := policytools.NewPolicyIterator(policies)
	if err != nil {
		log.Errorf("can not iterate over policies, err=%s", err)
		return
	}

	NumPolicyRules.Set(float64(0))

	for iterator.Next() {
		policy, target, peer, rule, direction := iterator.Items()

		// skip rules which don't have a valid target.
		// TODO filter blocks by current host to avoid unnecessary rules.
		if !valid(target) {
			log.Debugf("Target %s skipped for policy %s as invalid for the host", target, policy.ID)
			continue
		}

		// translates singe romana policy Rule into iptables chains.
		err := translateRule(
			policy,
			policytools.SchemePolicyOnTop,
			peer,
			target,
			rule,
			direction,
			iptables,
		)

		if err != nil {
			log.Errorf("Error appying %s policy to target %v and peer %v with rule %v, err=%s", direction, target, peer, rule, err)
			continue
		}

		NumPolicyRules.Inc()
	}
}

func cleanupUnusedChains(iptables *iptsave.IPtables, exec utilexec.Executable) {
	desiredFilter := iptables.TableByName("filter")

	// Load iptables rules from system.
	currentIPtables, err := LoadIPtables(exec)
	if err != nil {
		log.Errorf("Failed to load current iptables (%s), can not remove old chains", err)
		return
	}

	currentFilter := currentIPtables.TableByName("filter")
	if currentFilter == nil {
		log.Errorf("Failed to load current iptables (No filter table), can not remove old chains")
	}

	var romanaChainsInCurrentTables []string
	for _, currentChain := range currentFilter.Chains {
		if strings.HasPrefix(currentChain.Name, "ROMANA-") {
			romanaChainsInCurrentTables = append(romanaChainsInCurrentTables, currentChain.Name)
		}
	}

	for _, currentChain := range romanaChainsInCurrentTables {
		log.Tracef(5, "In cleanupUnusedChains, testing is %s exists in desired state", currentChain)
		desiredChain := desiredFilter.ChainByName(currentChain)
		if desiredChain == nil {
			log.Tracef(6, "In cleanupUnusedChains, scheduling chain %s for deletion", currentChain)
			desiredChain := iptsave.IPchain{Name: currentChain, Policy: "-", RenderState: iptsave.RenderDeleteRule}
			desiredFilter.Chains = append(desiredFilter.Chains, &desiredChain)
		}
	}
}

func EnsureTopRule(chain *iptsave.IPchain, rule *iptsave.IPrule) {
	if !chain.RuleInChain(rule) {
		chain.InsertRule(0, rule)
	}
}

func EnsureRules(baseChain *iptsave.IPchain, rules []*iptsave.IPrule) {
	for _, rule := range rules {
		if !baseChain.RuleInChain(rule) {
			InsertNormalRule(baseChain, rule)
		}
	}
}

func rules2list(rules ...*iptsave.IPrule) []*iptsave.IPrule {
	return rules
}

// translateRule translates specific combination of peer, target and rule into
// the set of iptables rules.
func translateRule(policy api.Policy,
	iptablesSchemeType string,
	peer, target api.Endpoint,
	rule api.Rule,
	direction string,
	iptables *iptsave.IPtables) error {

	// detect target and peer type to choose proper translation scheme.
	peerType := policytools.DetectPolicyPeerType(peer)
	dstType := policytools.DetectPolicyTargetType(target)

	// translationConfig is a schema that describes how to translate particular
	// combination of parameters.
	key := policytools.MakeBlueprintKey(direction, iptablesSchemeType, peerType, dstType)
	translationConfig, ok := policytools.Blueprints[key]
	if !ok {
		return errors.New("can't translate ... ")
	}

	// for now all our policies live in *filter table
	filter := iptables.TableByName("filter")

	// every combination of parameters will  be translated in 3 or 4 iptables
	// rules
	// rule 1-2) filters traffic by target - aka tenant owner of the policy
	// rule 3) filters traffic by peer - aka remote tenant, cidr, etc...
	// rule 4) filters traffic according to l4 protocol spec, and applies
	// a rule - aka ACCEPT/REJECT

	// first rule filters traffic for target tenant.
	baseChain := EnsureChainExists(filter, translationConfig.BaseChain)
	jumpFromBaseToPolicyRule := policytools.MakeRuleWithBody(
		translationConfig.TopRuleMatch(target, policy, direction), translationConfig.TopRuleAction(policy),
	)
	EnsureRules(baseChain, rules2list(jumpFromBaseToPolicyRule))

	// second rule filters traffic for target tenant (optional for SchemePolicyOnTop)
	secondBaseChainName := translationConfig.SecondBaseChain(policy)
	secondRuleMatch := translationConfig.SecondRuleMatch(target, policy, direction)
	secondRuleAction := translationConfig.SecondRuleAction(policy)
	if secondBaseChainName != "" && secondRuleMatch != "" && secondRuleAction != "" {
		secondBaseChain := EnsureChainExists(filter, secondBaseChainName)
		jumpFromSecondChainToThirdChainRule := policytools.MakeRuleWithBody(
			secondRuleMatch, secondRuleAction,
		)

		EnsureRules(secondBaseChain, rules2list(jumpFromSecondChainToThirdChainRule))

	}

	// third rule filters traffic by peer
	thirdBaseChainName := translationConfig.ThirdBaseChain(policy)
	thirdBaseChain := EnsureChainExists(filter, thirdBaseChainName)
	EnsureTopRule(thirdBaseChain, MakeIsolateRuleForChain())
	thirdRuleMatch := translationConfig.ThirdRuleMatch(peer, policy, direction)
	thirdRuleAction := translationConfig.ThirdRuleAction(policy)
	thirdRule := policytools.MakeRuleWithBody(
		thirdRuleMatch, thirdRuleAction,
	)
	EnsureRules(thirdBaseChain, rules2list(thirdRule))

	// fourth rule filters traffic by protocol spec.
	fourthBaseChainName := translationConfig.FourthBaseChain(policy)
	fourthBaseChain := EnsureChainExists(filter, fourthBaseChainName)
	fourthRuleAction := translationConfig.FourthRuleAction
	fourthRules := translationConfig.FourthRuleMatch(rule, fourthRuleAction)
	EnsureRules(fourthBaseChain, fourthRules)

	return nil
}

// targetValid validates that endpoint provided as a target refers to the known
// tenant and segment.
// Always true for non tenant types of matching.
func targetValid(target api.Endpoint, blocks []api.IPAMBlockResponse) bool {
	// This condition tells enforcer to always create policy that selects a label
	// even if no targets for that label could be found on current host,
	// Such strategy trades memory for cpu, if at any point we would want to
	// change that we will have to run labels evaluation here for each endpoint on
	// given host to check that policy is required.
	if len(target.Labels) > 0 {
		log.Debugf("target %s is valid because it selects labels", target)
		return true
	}

	// if endpoint doesn't match tenant this check is irrelevant.
	if target.TenantID == "" {
		log.Debugf("target %s is valid becuase it is not a tenant match", target)
		return true
	}

	// accumulate all known segments for this tenant.
	var segments []string
	for _, block := range blocks {

		log.Debugf("in targetValid comparing block.Tenant(%s) == target.TenantID(%s) = %t", block.Tenant, target.TenantID, block.Tenant == target.TenantID)

		if block.Tenant == target.TenantID {
			segments = append(segments, block.Segment)
		}
	}

	if len(segments) == 0 {
		log.Debugf("target %s is invalid because it matches no segments", target)
		return false
	}

	if target.SegmentID == "" {
		log.Debugf("target %s is valid because it has corresponding block and doesn't match any segment", target)
		return true
	}

	for _, segment := range segments {
		log.Debugf("in targetValid comparing target.SegmentID(%s) == segment(%s) = %t", target.SegmentID, segment, target.SegmentID == segment)

		if target.SegmentID == segment {
			return true
		}
	}

	return false
}
