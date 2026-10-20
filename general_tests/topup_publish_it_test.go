//go:build integration
// +build integration

// Copyright ITsysCOM GmbH
// SPDX-License-Identifier: AGPL-3.0-or-later

package general_tests

import (
	"testing"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func TestTopUpPublish(t *testing.T) {
	switch *utils.DBType {
	case utils.MetaInternal:
	case utils.MetaMySQL, utils.MetaMongo, utils.MetaPostgres:
		t.SkipNow()
	default:
		t.Fatal("unsupported dbtype value")
	}

	content := `{

"general": {
	"log_level": 7,
},

"data_db": {								
	"db_type": "*internal"
},

"stor_db": {
	"db_type": "*internal"
},

"rals": {
	"enabled": true,
	"thresholds_conns": ["*localhost"],
},

"thresholds": {
	"enabled": true,
	"store_interval": "-1",
},

"schedulers": {
	"enabled": true,
},

"apiers": {
	"enabled": true,
	"scheduler_conns": ["*internal"]
}
}`

	ng := engine.TestEngine{
		ConfigJSON: content,
	}
	client, _ := ng.Run(t)

	t.Run("CreateThreshold", func(t *testing.T) {
		thdPrf := &engine.ThresholdProfileWithAPIOpts{
			ThresholdProfile: &engine.ThresholdProfile{
				Tenant:    "cgrates.org",
				ID:        "LogOnAnyEvent",
				ActionIDs: []string{utils.MetaLog},
				MinHits:   1,
				MaxHits:   1,
				Async:     true,
			},
		}
		var reply string
		if err := client.Call(context.Background(), utils.APIerSv1SetThresholdProfile, thdPrf, &reply); err != nil {
			t.Error(err)
		} else if reply != utils.OK {
			t.Error("Unexpected reply returned", reply)
		}
	})

	t.Run("CreateAction", func(t *testing.T) {
		var reply string
		topupAction := &utils.AttrSetActions{ActionsId: "TOPUP_ACTION", Actions: []*utils.TPAction{
			{
				Identifier:  utils.MetaTopUp,
				BalanceId:   "BalID1",
				BalanceType: utils.MetaMonetary,
				Units:       "5",
			},
			// {
			// 	Identifier: utils.MetaPublishAccount,
			// },
		}}

		if err := client.Call(context.Background(), utils.APIerSv2SetActions, topupAction, &reply); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("CreateActionPlan", func(t *testing.T) {
		atms := &engine.AttrSetActionPlan{
			Id: "PACKAGE_1",
			ActionPlan: []*engine.AttrActionPlan{
				{
					ActionsId: "TOPUP_ACTION",
					Time:      utils.MetaASAP,
					TimingID:  utils.MetaASAP,
				},
			},
		}

		var reply string
		if err := client.Call(context.Background(), utils.APIerSv1SetActionPlan, atms, &reply); err != nil {
			t.Fatal(err)
		} else if reply != utils.OK {
			t.Fatalf("received: %s", reply)
		}
	})

	t.Run("CreateAccount", func(t *testing.T) {
		attrSetAcnt := engine.AttrSetAccount{
			Tenant:          "cgrates.org",
			Account:         "1001",
			ActionPlanIDs:   []string{"PACKAGE_1"},
			ReloadScheduler: true,
		}
		var reply string
		if err := client.Call(context.Background(), utils.APIerSv2SetAccount, &attrSetAcnt, &reply); err != nil {
			t.Error(err)
		}
	})

	t.Run("CheckAccount", func(t *testing.T) {
		argGet := &utils.AttrGetAccount{
			Tenant:  "cgrates.org",
			Account: "1001",
		}
		var acnt *engine.Account
		if err := client.Call(context.Background(), utils.APIerSv2GetAccount, argGet, &acnt); err != nil {
			t.Error(err)
		}

		t.Log(utils.ToJSON(acnt))
	})

	t.Run("TopUpAccount", func(t *testing.T) {
		attrSetAcnt := engine.AttrSetAccount{
			Tenant:          "cgrates.org",
			Account:         "1001",
			ActionPlanIDs:   []string{"PACKAGE_1"},
			ReloadScheduler: true,
		}
		var reply string
		if err := client.Call(context.Background(), utils.APIerSv2SetAccount, &attrSetAcnt, &reply); err != nil {
			t.Error(err)
		}
	})

	t.Run("CheckAccount", func(t *testing.T) {
		argGet := &utils.AttrGetAccount{
			Tenant:  "cgrates.org",
			Account: "1001",
		}
		var acnt *engine.Account
		if err := client.Call(context.Background(), utils.APIerSv2GetAccount, argGet, &acnt); err != nil {
			t.Error(err)
		}

		t.Log(utils.ToJSON(acnt))
	})

}
