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

package config

import (
	"slices"
	"strings"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/utils"
)

// MigratorCgrCfg the migrator config section
type MigratorCgrCfg struct {
	OutDataDBEncoding string
	UsersFilters      []string
	InItems           map[string]*InItem // contains the in items as the keys of the map, and the DataDB ids of each item in InItems
	OutDataDBOpts     *DataDBOpts
}

// InItem contains the DataDB ids of each item
type InItem struct {
	DataDBID string // ID of the DataDB connection that this item belongs to
}

// loadFromJSONCfg loads Database config from JsonCfg
func (iI *InItem) loadFromJSONCfg(jsonII *InItemJson) (err error) {
	if jsonII == nil {
		return
	}
	if jsonII.Datadb_id != nil {
		iI.DataDBID = *jsonII.Datadb_id
	}
	return
}

// Clone returns the cloned object
func (iI *InItem) Clone() *InItem {
	if iI == nil {
		return nil
	}
	return &InItem{
		DataDBID: iI.DataDBID,
	}
}

// AsMapInterface returns the config as a map[string]any
func (iI *InItem) AsMapInterface() (initialMP map[string]any) {
	initialMP = map[string]any{
		utils.DataDBIDCfg: iI.DataDBID,
	}
	return
}

// loadMigratorCgrCfg loads the Migrator section of the configuration
func (mg *MigratorCgrCfg) Load(ctx *context.Context, jsnCfg ConfigDB, _ *CGRConfig) (err error) {
	jsnMigratorCgrCfg := new(MigratorCfgJson)
	if err = jsnCfg.GetSection(ctx, MigratorJSON, jsnMigratorCgrCfg); err != nil {
		return
	}
	return mg.loadFromJSONCfg(jsnMigratorCgrCfg)
}

func (mg *MigratorCgrCfg) loadFromJSONCfg(jsnCfg *MigratorCfgJson) (err error) {
	if jsnCfg == nil {
		return
	}
	if jsnCfg.Out_dataDB_encoding != nil {
		mg.OutDataDBEncoding = strings.TrimPrefix(*jsnCfg.Out_dataDB_encoding, "*")
	}
	if jsnCfg.Users_filters != nil && len(*jsnCfg.Users_filters) != 0 {
		mg.UsersFilters = slices.Clone(*jsnCfg.Users_filters)
	}
	if jsnCfg.In_items != nil {
		for kJsn, vJsn := range jsnCfg.In_items {
			val, has := mg.InItems[kJsn]
			if val == nil || !has {
				val = new(InItem)
			}
			if err = val.loadFromJSONCfg(vJsn); err != nil {
				return
			}
			mg.InItems[kJsn] = val
		}
	}
	if jsnCfg.Out_dataDB_opts != nil {
		err = mg.OutDataDBOpts.loadFromJSONCfg(jsnCfg.Out_dataDB_opts)
	}
	return
}

// AsMapInterface returns the config as a map[string]any
func (mg MigratorCgrCfg) AsMapInterface() any {
	outDataDBOpts := map[string]any{
		utils.RedisMaxConnsCfg:           mg.OutDataDBOpts.RedisMaxConns,
		utils.RedisConnectAttemptsCfg:    mg.OutDataDBOpts.RedisConnectAttempts,
		utils.RedisSentinelNameCfg:       mg.OutDataDBOpts.RedisSentinel,
		utils.RedisClusterCfg:            mg.OutDataDBOpts.RedisCluster,
		utils.RedisClusterSyncCfg:        mg.OutDataDBOpts.RedisClusterSync.String(),
		utils.RedisClusterOnDownDelayCfg: mg.OutDataDBOpts.RedisClusterOndownDelay.String(),
		utils.RedisConnectTimeoutCfg:     mg.OutDataDBOpts.RedisConnectTimeout.String(),
		utils.RedisReadTimeoutCfg:        mg.OutDataDBOpts.RedisReadTimeout.String(),
		utils.RedisWriteTimeoutCfg:       mg.OutDataDBOpts.RedisWriteTimeout.String(),
		utils.RedisPoolPipelineWindowCfg: mg.OutDataDBOpts.RedisPoolPipelineWindow.String(),
		utils.RedisPoolPipelineLimitCfg:  mg.OutDataDBOpts.RedisPoolPipelineLimit,
		utils.RedisTLSCfg:                mg.OutDataDBOpts.RedisTLS,
		utils.RedisClientCertificateCfg:  mg.OutDataDBOpts.RedisClientCertificate,
		utils.RedisClientKeyCfg:          mg.OutDataDBOpts.RedisClientKey,
		utils.RedisCACertificateCfg:      mg.OutDataDBOpts.RedisCACertificate,
		utils.MongoQueryTimeoutCfg:       mg.OutDataDBOpts.MongoQueryTimeout.String(),
		utils.MongoConnSchemeCfg:         mg.OutDataDBOpts.MongoConnScheme,
	}
	var items map[string]any
	if mg.InItems != nil {
		items = make(map[string]any)
		for itemID, item := range mg.InItems {
			items[itemID] = item.AsMapInterface()
		}
	}
	return map[string]any{
		utils.OutDataDBEncodingCfg: mg.OutDataDBEncoding,
		utils.InItemsCfg:           items,
		utils.OutDataDBOptsCfg:     outDataDBOpts,
		utils.UsersFiltersCfg:      slices.Clone(mg.UsersFilters),
	}
}

func (MigratorCgrCfg) SName() string            { return MigratorJSON }
func (mg MigratorCgrCfg) CloneSection() Section { return mg.Clone() }

// Clone returns a deep copy of MigratorCgrCfg
func (mg MigratorCgrCfg) Clone() (cln *MigratorCgrCfg) {
	cln = &MigratorCgrCfg{
		OutDataDBEncoding: mg.OutDataDBEncoding,
		InItems:           make(map[string]*InItem),
		OutDataDBOpts:     mg.OutDataDBOpts.Clone(),
	}
	for k, v := range mg.InItems {
		cln.InItems[k] = v.Clone()
	}
	if mg.UsersFilters != nil {
		cln.UsersFilters = slices.Clone(mg.UsersFilters)
	}
	return
}

type MigratorCfgJson struct {
	Out_dataDB_encoding *string
	Users_filters       *[]string
	In_items            map[string]*InItemJson
	Out_dataDB_opts     *DBOptsJson
}

type InItemJson struct {
	Datadb_id *string
}

func (iI *InItem) Equals(itm2 *InItem) bool {
	return iI == nil && itm2 == nil ||
		iI != nil && itm2 != nil && iI.DataDBID == itm2.DataDBID
}

func diffInItemJson(d *InItemJson, v1, v2 *InItem) *InItemJson {
	if d == nil {
		d = new(InItemJson)
	}
	if v2.DataDBID != v1.DataDBID {
		d.Datadb_id = utils.StringPointer(v2.DataDBID)
	}
	return d
}

func diffMapInItemJson(d map[string]*InItemJson, v1 map[string]*InItem,
	v2 map[string]*InItem) map[string]*InItemJson {
	if d == nil {
		d = make(map[string]*InItemJson)
	}
	for k, val2 := range v2 {
		if val1, has := v1[k]; !has {
			d[k] = diffInItemJson(d[k], new(InItem), val2)
		} else if !val1.Equals(val2) {
			d[k] = diffInItemJson(d[k], val1, val2)
		}
	}
	return d
}

func diffMigratorCfgJson(d *MigratorCfgJson, v1, v2 *MigratorCgrCfg) *MigratorCfgJson {
	if d == nil {
		d = new(MigratorCfgJson)
	}
	if v1.OutDataDBEncoding != v2.OutDataDBEncoding {
		d.Out_dataDB_encoding = utils.StringPointer(v2.OutDataDBEncoding)
	}

	if !slices.Equal(v1.UsersFilters, v2.UsersFilters) {
		d.Users_filters = utils.SliceStringPointer(slices.Clone(v2.UsersFilters))
	}
	d.In_items = diffMapInItemJson(d.In_items, v1.InItems, v2.InItems)
	d.Out_dataDB_opts = diffDataDBOptsJsonCfg(d.Out_dataDB_opts, v1.OutDataDBOpts, v2.OutDataDBOpts)
	return d
}
