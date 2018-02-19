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

package listener

import (
	"reflect"
	"time"

	"github.com/romana/core/common/log/trace"
	log "github.com/romana/rlog"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

const (
	// TODO make a parameter. Stas.
	processorTickTime = 4
)

// process is a goroutine that consumes resource update events coming from
// Kubernetes and:
// 1. On receiving an added or deleted event:
//    i. add it to the queue
//    ii. on a timer event, send the events to handleNetworkPolicyEvents and empty the queue
// 2. On receiving a done event, exit the goroutine
func (l *KubeListener) process(in <-chan Event, done chan struct{}) {
	log.Debugf("KubeListener: process(): Entered with in %v, done %v", in, done)

	timer := time.NewTicker(processorTickTime * time.Second)
	var networkPolicyEvents []Event

	go func() {
		for {
			select {
			case <-timer.C:
				if len(networkPolicyEvents) > 0 {
					log.Infof("Calling network policy handler for scheduled %d events", len(networkPolicyEvents))
					handleNetworkPolicyEvents(networkPolicyEvents, l)
					networkPolicyEvents = nil
				}
			case e := <-in:
				log.Debugf("KubeListener: process(): Got %v", e)
				switch obj := e.Object.(type) {
				case *v1beta1.NetworkPolicy:
					log.Tracef(trace.Inside, "Scheduing network policy action, now scheduled %d actions", len(networkPolicyEvents))
					networkPolicyEvents = append(networkPolicyEvents, e)
				case *v1.Namespace:
					log.Tracef(trace.Inside, "Processor received namespace")
					handleNamespaceEvent(e, l)
				default:
					log.Errorf("Processor received an event of unkonwn type %s, ignoring object %s", reflect.TypeOf(obj), obj)
				}
			case <-done:
				log.Debugf("KubeListener: process(): Got done")
				timer.Stop()
				return
			}
		}
	}()
}
