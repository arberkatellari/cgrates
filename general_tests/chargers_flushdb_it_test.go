//go:build integration
// +build integration

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

package general_tests

import (
	"fmt"
	"testing"

	"github.com/cgrates/birpc/context"
	v1 "github.com/cgrates/cgrates/apier/v1"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func TestFlushDBChargers(t *testing.T) {

	content := `{
// CGRateS Configuration file
//


"general": {
	"log_level": 7,
	"reply_timeout": "50s",
},


"listen": {
	"rpc_json": ":2012",
	"rpc_gob": ":2013",
	"http": ":2080",
},

"data_db": {								// database used to store runtime data (eg: accounts, cdr stats)
	"db_type": "redis",						// data_db type: <redis|mongo>
	"db_port": 6379, 						// data_db port to reach the database
	"db_name": "10", 						// data_db database name to connect to
},

"stor_db": {
	"db_password": "CGRateS.org",
},


"rals": {
	"enabled": true,
	"thresholds_conns": ["*internal"],
	"max_increments":3000000,
},


"schedulers": {
	"enabled": true,
	"cdrs_conns": ["*internal"],
	"stats_conns": ["*localhost"],
},


"cdrs": {
	"enabled": true,
	"chargers_conns":["*internal"],
},


"attributes": {
	"enabled": true,
	"stats_conns": ["*localhost"],
	"resources_conns": ["*localhost"],
	"apiers_conns": ["*localhost"]
},


"chargers": {
	"enabled": true,
	"attributes_conns": ["*internal"],
},


"resources": {
	"enabled": true,
	"store_interval": "1s",
	"thresholds_conns": ["*internal"]
},


"stats": {
	"enabled": true,
	"store_interval": "1s",
	"thresholds_conns": ["*internal"],
},


"thresholds": {
	"enabled": true,
	"store_interval": "1s",
},


"routes": {
	"enabled": true,
	"prefix_indexed_fields":["*req.Destination"],
	"stats_conns": ["*internal"],
	"resources_conns": ["*internal"],
	"rals_conns": ["*internal"],
},


"sessions": {
	"enabled": true,
	"routes_conns": ["*internal"],
	"resources_conns": ["*internal"],
	"attributes_conns": ["*internal"],
	"rals_conns": ["*internal"],
	"cdrs_conns": ["*localhost"],
	"chargers_conns": ["*internal"],
},


"migrator":{
	"out_stordb_password": "CGRateS.org",
	"users_filters":["Account"],
},


"apiers": {
	"enabled": true,
	"scheduler_conns": ["*internal"],
},


"filters": {
	"stats_conns": ["*localhost"],
	"resources_conns": ["*internal"],
	"apiers_conns": ["*internal"],
},


}
`

	testEnv := TestEnvironment{
		Name:       "TestFlushDBChargers",
		ConfigJSON: content,
	}
	client, cfg := testEnv.Setup(t, *utils.WaitRater)

	t.Run("SetCharger", func(t *testing.T) {
		chargerProfile := &v1.ChargerWithAPIOpts{
			ChargerProfile: &engine.ChargerProfile{
				Tenant:       "cgrates.org",
				ID:           "Charger_API_Default",
				RunID:        "*Charger_API_Default_RunID",
				AttributeIDs: []string{"*none"},
				Weight:       20,
			},
		}
		var result string
		if err := client.Call(context.Background(), utils.APIerSv1SetChargerProfile, chargerProfile, &result); err != nil {
			t.Fatal(err)
		}

		var reply *engine.ChargerProfiles
		if err := client.Call(context.Background(), utils.ChargerSv1GetChargersForEvent,
			utils.CGREvent{}, &reply); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("First cfi get <%+v>\n", utils.ToJSON(reply))
	})

	t.Run("Flushdb", func(t *testing.T) {
		flushDBs(t, cfg, true, true)
		fmt.Println("FlushedDB")
	})

	t.Run("SetChargerAgain", func(t *testing.T) {
		chargerProfile := &v1.ChargerWithAPIOpts{
			ChargerProfile: &engine.ChargerProfile{
				Tenant:       "cgrates.org",
				ID:           "Charger_API_Default",
				RunID:        "*Charger_API_Default_RunID",
				AttributeIDs: []string{"*none"},
				Weight:       20,
			},
		}
		var result string
		if err := client.Call(context.Background(), utils.APIerSv1SetChargerProfile, chargerProfile, &result); err != nil {
			t.Fatal(err)
		}

		var reply *engine.ChargerProfiles
		if err := client.Call(context.Background(), utils.ChargerSv1GetChargersForEvent,
			utils.CGREvent{}, &reply); err.Error() == utils.ErrNotFound.Error() {
			t.Log("cfi not found, err: ", err)
		}
		fmt.Printf("2nd cfi get <%+v>\n", utils.ToJSON(reply))

		var rplyIDs []string
		if err := client.Call(context.Background(), utils.APIerSv1GetChargerProfileIDs, &utils.PaginatorWithTenant{}, &rplyIDs); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		fmt.Printf("cpp get <%+v>\n", utils.ToJSON(rplyIDs))
	})

}
