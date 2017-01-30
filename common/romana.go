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
)

// RomanaEntity represents a principal entity of Romana. They have the following
// fields:
// - UUID -- unique ID within Romana.
// - ExternalID -- unique ID within some external system.
// - NetworkID -- generated sequential ID, essential to Romana addressing scheme.
type RomanaEntity interface {
	GetKind() string
	GetUUID() string
	Bytes() ([]byte)

	// TODO for Stas temporary due to ip can't be assigned outside of storage,
	// move ID allocation outside of storage and remove this method from the interface
	// RomanaIP must be set in Topology context instead.
	SetRomanaIP(string)
	GetRomanaIP() string

	// TODO for Stas temporary to allow storage to set ID, storage must not do this.
	SetID(uint64)
	GetID() uint64
}
