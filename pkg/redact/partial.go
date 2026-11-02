// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redact

import (
	"errors"
	"net/netip"
)

// KeepIPv4Octets retains the last one or two octets while replacing the prefix
// with 10.0.0 for one octet or a synthetic prefix for two octets.
// Different originals may share a replacement when their retained suffixes
// match. Unchanged one-octet addresses still fail before data is changed.
func (r *Redactor) KeepIPv4Octets(count int) error {
	if count != 1 && count != 2 {
		return errors.New("keep-ipv4-octets must be 1 or 2")
	}
	if r.logIPv4QuestionMark {
		return errors.New("keep-ipv4-octets and hide-log-ipv4 are mutually exclusive")
	}
	proposed := make(map[netip.Addr]netip.Addr, len(r.mapping))
	destinations := make(map[netip.Addr]bool)
	for original, replacement := range r.mapping {
		if original.Is4() {
			bytes := original.As4()
			tail := [2]byte{0, bytes[3]}
			if count == 2 {
				tail[0] = bytes[2]
			}
			replacement = netip.AddrFrom4([4]byte{198, 18, tail[0], tail[1]})
			if count == 1 {
				replacement = netip.AddrFrom4([4]byte{10, 0, 0, tail[1]})
				if replacement == original {
					return errors.New("IPv4 already uses the fixed 10.0.0 prefix: use full replacement IPs with ip-map-output")
				}
			} else if replacement == original {
				replacement = netip.AddrFrom4([4]byte{198, 19, tail[0], tail[1]})
			}
		}
		proposed[original] = replacement
		destinations[replacement] = true
	}
	r.mapping = proposed
	r.destinations = destinations
	r.keptIPv4Octets = count
	return nil
}
