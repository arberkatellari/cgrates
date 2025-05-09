/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package engine

import (
	"sort"

	"github.com/cgrates/cgrates/utils"
)

// ChargerProfile is the config for one Charger
type ChargerProfile struct {
	Tenant             string
	ID                 string
	FilterIDs          []string
	ActivationInterval *utils.ActivationInterval // Activation interval
	RunID              string
	AttributeIDs       []string // perform data aliasing based on these Attributes
	Weight             float64
}

// Clone method for ChargerProfile
func (cp *ChargerProfile) Clone() *ChargerProfile {
	if cp == nil {
		return nil
	}
	clone := &ChargerProfile{
		Tenant: cp.Tenant,
		ID:     cp.ID,
		RunID:  cp.RunID,
		Weight: cp.Weight,
	}
	if cp.FilterIDs != nil {
		clone.FilterIDs = make([]string, len(cp.FilterIDs))
		copy(clone.FilterIDs, cp.FilterIDs)
	}
	if cp.AttributeIDs != nil {
		clone.AttributeIDs = make([]string, len(cp.AttributeIDs))
		copy(clone.AttributeIDs, cp.AttributeIDs)
	}
	if cp.ActivationInterval != nil {
		clone.ActivationInterval = cp.ActivationInterval.Clone()
	}
	return clone
}

// CacheClone returns a clone of ChargerProfile used by ltcache CacheCloner
func (cp *ChargerProfile) CacheClone() any {
	return cp.Clone()
}

// ChargerProfileWithAPIOpts is used in replicatorV1 for dispatcher
type ChargerProfileWithAPIOpts struct {
	*ChargerProfile
	APIOpts map[string]any
}

func (cP *ChargerProfile) TenantID() string {
	return utils.ConcatenatedKey(cP.Tenant, cP.ID)
}

// ChargerProfiles is a sortable list of Charger profiles
type ChargerProfiles []*ChargerProfile

// Sort is part of sort interface, sort based on Weight
func (cps ChargerProfiles) Sort() {
	sort.Slice(cps, func(i, j int) bool { return cps[i].Weight > cps[j].Weight })
}
