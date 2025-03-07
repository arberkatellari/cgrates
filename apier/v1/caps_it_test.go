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

package v1

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgrates/birpc"
	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

var (
	capsCfgPath   string
	capsCfg       *config.CGRConfig
	capsRPC       *birpc.Client
	capsBiRPC     *birpc.BirpcClient
	capsConfigDIR string //run tests for specific configuration

	sTestsCaps = []func(t *testing.T){
		testCapsInitCfg,
		testCapsStartEngine,
		testCapsRPCConn,
		testCapsBusyAPIs,
		testCapsQueueAPIs,
		testCapsOnHTTPBusy,
		testCapsOnHTTPQueue,
		testCapsOnBiJSONBusy,
		testCapsOnBiJSONQueue,
		testCapsKillEngine,
	}

	// used by benchmarks
	capsOnce       sync.Once
	capsLastCfgDir string
)

// Test start here
func TestCapsBusyJSON(t *testing.T) {
	capsConfigDIR = "caps_busy"
	for _, stest := range sTestsCaps {
		t.Run(capsConfigDIR, stest)
	}
}

func TestCapsQueueJSON(t *testing.T) {
	capsConfigDIR = "caps_queue"
	for _, stest := range sTestsCaps {
		t.Run(capsConfigDIR, stest)
	}
}

func testCapsInitCfg(t *testing.T) {
	var err error
	capsCfgPath = path.Join(*utils.DataDir, "conf", "samples", capsConfigDIR)
	capsCfg, err = config.NewCGRConfigFromPath(capsCfgPath)
	if err != nil {
		t.Error(err)
	}
}

// Start CGR Engine
func testCapsStartEngine(t *testing.T) {
	if _, err := engine.StopStartEngine(capsCfgPath, *utils.WaitRater); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func testCapsRPCConn(t *testing.T) {
	var err error
	capsRPC, err = newRPCClient(capsCfg.ListenCfg()) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal(err)
	}
	if capsBiRPC, err = utils.NewBiJSONrpcClient(capsCfg.SessionSCfg().ListenBijson,
		nil); err != nil {
		t.Fatal(err)
	}
}

func testCapsBusyAPIs(t *testing.T) {
	if capsConfigDIR != "caps_busy" {
		t.SkipNow()
	}
	var failedAPIs int
	wg := new(sync.WaitGroup)
	lock := new(sync.Mutex)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp string
			if err := capsRPC.Call(context.Background(), utils.CoreSv1Sleep,
				&utils.DurationArgs{Duration: 10 * time.Millisecond},
				&resp); err != nil {
				lock.Lock()
				failedAPIs++
				lock.Unlock()
				return
			}
		}()
	}
	wg.Wait()
	if failedAPIs < 2 {
		t.Errorf("Expected at leat 2 APIs to wait")
	}
}

func testCapsQueueAPIs(t *testing.T) {
	if capsConfigDIR != "caps_queue" {
		t.SkipNow()
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp string
			if err := capsRPC.Call(context.Background(), utils.CoreSv1Sleep,
				&utils.DurationArgs{Duration: 10 * time.Millisecond},
				&resp); err != nil {
				t.Error(err)
				return
			}
		}()
	}
	wg.Wait()
}

func testCapsOnHTTPBusy(t *testing.T) {
	if capsConfigDIR != "caps_busy" {
		t.SkipNow()
	}
	var fldAPIs int64
	wg := new(sync.WaitGroup)
	lock := new(sync.Mutex)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			resp, err := http.Post("http://localhost:2080/jsonrpc", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"method": "CoreSv1.Sleep", "params": [{"Duration":10000000}], "id":%d}`, index))))
			if err != nil {
				t.Error(err)
				return
			}
			contents, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
			if strings.Contains(string(contents), utils.ErrMaxConcurrentRPCExceeded.Error()) {
				lock.Lock()
				fldAPIs++
				lock.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if fldAPIs < 2 {
		t.Errorf("Expected at leat 2 APIs to wait")
	}
}

func testCapsOnHTTPQueue(t *testing.T) {
	if capsConfigDIR != "caps_queue" {
		t.SkipNow()
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := http.Post("http://localhost:2080/jsonrpc", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"method": "CoreSv1.Sleep", "params": [{"Duration":10000000}], "id":%d}`, index))))
			if err != nil {
				t.Error(err)
				return
			}
		}(i)
	}
	wg.Wait()
}

func testCapsOnBiJSONBusy(t *testing.T) {
	if capsConfigDIR != "caps_busy" {
		t.SkipNow()
	}
	var failedAPIs int
	lock := new(sync.Mutex)
	errChan := make(chan error)   // to retrieve and verify api errors
	waitCh := make(chan struct{}) // helper channel to break the for-select loop
	go func() {
		wg := new(sync.WaitGroup)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var resp string
				if err := capsBiRPC.Call(context.Background(), utils.SessionSv1Sleep,
					&utils.DurationArgs{
						Duration: 10 * time.Millisecond,
					}, &resp); err != nil {
					errChan <- err
					lock.Lock()
					failedAPIs++
					lock.Unlock()
					return
				}
			}()
		}
		wg.Wait()
		close(waitCh)
	}()
	waiting := true
	for waiting {
		select {
		case err := <-errChan:
			if err.Error() != utils.ErrMaxConcurrentRPCExceeded.Error() {
				t.Errorf("expected: <%+v>, \nreceived: <%+v>",
					utils.ErrMaxConcurrentRPCExceeded, err)
			}
		case <-waitCh:
			waiting = false
		}

	}
	if failedAPIs < 2 {
		t.Errorf("Expected at least 2 APIs to wait")
	}
}

func testCapsOnBiJSONQueue(t *testing.T) {
	if capsConfigDIR != "caps_queue" {
		t.SkipNow()
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp string
			if err := capsBiRPC.Call(context.Background(), utils.SessionSv1Sleep,
				&utils.DurationArgs{Duration: 10 * time.Millisecond},
				&resp); err != nil {
				t.Error(err)
				return
			}
		}()
	}
	wg.Wait()
}

func testCapsKillEngine(t *testing.T) {
	if err := engine.KillEngine(100); err != nil {
		t.Error(err)
	}
}

func benchmarkInit(b *testing.B, cfgDir string) {
	b.StopTimer()
	// restart cgrates only if needed
	if cfgDir != capsLastCfgDir {
		capsOnce = sync.Once{}
	}
	capsOnce.Do(func() {
		capsLastCfgDir = cfgDir
		var err error
		capsCfgPath = path.Join(*utils.DataDir, "conf", "samples", cfgDir)
		if capsCfg, err = config.NewCGRConfigFromPath(capsCfgPath); err != nil {
			b.Fatal(err)
		}
		if _, err := engine.StopStartEngine(capsCfgPath, *utils.WaitRater); err != nil {
			b.Fatal(err)
		}
		if capsRPC, err = newRPCClient(capsCfg.ListenCfg()); err != nil {
			b.Fatal(err)
		}
		// b.Logf("Preparation done for %s", cfgDir)
	})
	b.StartTimer()
}

func benchmarkCall(b *testing.B) {
	var rply map[string]any
	for i := 0; i < b.N; i++ {
		if err := capsRPC.Call(context.Background(), utils.CoreSv1Status, &utils.TenantWithAPIOpts{}, &rply); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkCapsWithLimit(b *testing.B) {
	benchmarkInit(b, "caps_queue_bench")
	benchmarkCall(b)
}

func BenchmarkCapsWithoutLimit(b *testing.B) {
	benchmarkInit(b, "tutmysql")
	benchmarkCall(b)
}
