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

package services

import (
	"fmt"
	"sync"

	"github.com/cgrates/cgrates/agents"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/servmanager"
	"github.com/cgrates/cgrates/utils"
)

// NewDNSAgent returns the DNS Agent
func NewDNSAgent(cfg *config.CGRConfig, filterSChan chan *engine.FilterS,
	shdChan *utils.SyncedChan, connMgr *engine.ConnManager,
	srvDep map[string]*sync.WaitGroup) servmanager.Service {
	return &DNSAgent{
		cfg:         cfg,
		filterSChan: filterSChan,
		shdChan:     shdChan,
		connMgr:     connMgr,
		srvDep:      srvDep,
	}
}

// DNSAgent implements Agent interface
type DNSAgent struct {
	sync.RWMutex
	cfg         *config.CGRConfig
	filterSChan chan *engine.FilterS
	shdChan     *utils.SyncedChan

	stopChan    chan struct{} // used as signal to start shutting down all DNS listeners
	shdComplete chan struct{} // used as signal to indicate all shutting down is completed

	dns     *agents.DNSAgent
	connMgr *engine.ConnManager
	srvDep  map[string]*sync.WaitGroup
}

// Start should handle the service start
func (dns *DNSAgent) Start() (err error) {
	if dns.IsRunning() {
		return utils.ErrServiceAlreadyRunning
	}
	filterS := <-dns.filterSChan
	dns.filterSChan <- filterS
	dns.Lock()
	defer dns.Unlock()
	dns.dns = agents.NewDNSAgent(dns.cfg, filterS, dns.connMgr)

	dns.stopChan = make(chan struct{})
	dns.shdComplete = make(chan struct{})
	go dns.listenAndServe(dns.stopChan, dns.shdComplete)

	return
}

// Reload handles the change of config
func (dns *DNSAgent) Reload() (err error) {
	filterS := <-dns.filterSChan
	dns.filterSChan <- filterS

	dns.Lock()
	defer dns.Unlock()

	if err := dns.Shutdown(); err != nil {
		return err
	}

	dns.dns = agents.NewDNSAgent(dns.cfg, filterS, dns.connMgr)

	dns.stopChan = make(chan struct{})
	dns.shdComplete = make(chan struct{})
	go dns.listenAndServe(dns.stopChan, dns.shdComplete)
	return
}

func (dns *DNSAgent) listenAndServe(stopChan chan struct{}, shdComplete chan struct{}) (err error) {
	dns.dns.RLock()
	defer dns.dns.RUnlock()
	if err = dns.dns.ListenAndServe(stopChan, shdComplete); err != nil {
		utils.Logger.Err(fmt.Sprintf("<%s> error: <%s>", utils.DNSAgent, err.Error()))
		dns.shdChan.CloseOnce() // stop the engine here
		// dns.dns = nil // Add after services revamp
	}
	return
}

// Shutdown stops the service
func (dns *DNSAgent) Shutdown() (err error) {
	if dns.dns == nil {
		return
	}
	if dns.dns.LASisON { // Has to stay to trigger srvMngr.shdWg.Done() in case of error at dns.dns.ListenAndServe
		close(dns.stopChan) // start shutting all dns servers at the same time
		<-dns.shdComplete   // wait for all listeners to be shut
	}

	dns.dns.Lock()
	defer dns.dns.Unlock()
	dns.dns = nil
	return
}

// IsRunning returns if the service is running
func (dns *DNSAgent) IsRunning() bool {
	dns.RLock()
	defer dns.RUnlock()
	return dns.dns != nil
}

// ServiceName returns the service name
func (dns *DNSAgent) ServiceName() string {
	return utils.DNSAgent
}

// ShouldRun returns if the service should be running
func (dns *DNSAgent) ShouldRun() bool {
	return dns.cfg.DNSAgentCfg().Enabled
}
