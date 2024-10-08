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
package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/cgrates/cgrates/utils"
	"github.com/mediocregopher/radix/v3"
)

var (
	sTestsOnStorIT = []func(t *testing.T){
		testOnStorITFlush,
	}
)

func TestOnStorIT(t *testing.T) {
	for _, stest := range sTestsOnStorIT {
		t.Run(*utils.DBType, stest)
	}
}

func testOnStorITFlush(t *testing.T) {
	dDB, err := NewRedisStorage(
		"127.0.0.1:6379",
		10, "cgrates", "", "msgpack",
		10, 20, "", false, 5*time.Second, 0, 0, 0, 0, 150*time.Microsecond, 0, false, utils.EmptyString, utils.EmptyString, utils.EmptyString)
	if err != nil {
		t.Fatal("Could not connect to Redis", err.Error())
	}
	var totalStored int
	for totalStored < 2*1024*1024*1024 { // 2 gb total data size to generate
		data := make([]byte, 512*1024) // 512 kb
		pattern := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := range data {
			data[i] = pattern[i%len(pattern)]
		}
		key := fmt.Sprintf("%s%d", "testdata-", totalStored)
		err := dDB.client.Do(radix.Cmd(nil, "SET", key, string(data)))
		if err != nil {
			t.Fatalf("Error storing data in Redis: %v", err)
		}
		totalStored += len(data)
		fmt.Printf("Stored %d bytes of data\nTotalStored %d\n", len(data), totalStored)
	}
}
