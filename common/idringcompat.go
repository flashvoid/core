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
package common

import (
	"encoding/json"
)

// func (idRing *IDRing) ReclaimID(id uint64) error {
func (idRing *RingWrapper) GetName() string { return "Not implemented" }
func (idRing *RingWrapper) GetUUID() string { return "Not implemented" }
func (idRing *RingWrapper) GetKind() string { return idRing.IR.Kind }
func (idRing *RingWrapper) Bytes() []byte {
	b, _ := json.Marshal(idRing.IR)
	return b
}
func (idRing *RingWrapper) GetRomanaIP() string { return "Not Implemented" }
func (idRing *RingWrapper) SetRomanaIP(ip string) { return }
// func (idRing *RingWrapper) GetID() uint64 { return 0 }
func (idRing *RingWrapper) GetID() (uint64) { return 0 }
func (idRing *RingWrapper) SetID(id uint64) { return }

// TODO for Stas, temporary wrapper to pass idring to storage,
// RomanaEntity methods must be defined on IDRing itself.
type RingWrapper struct {
	IR IDRing
}
