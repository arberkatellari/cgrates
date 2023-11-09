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

package agents

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/miekg/dns"
)

// NewDNSAgent is the constructor for DNSAgent
func NewDNSAgent(cgrCfg *config.CGRConfig, fltrS *engine.FilterS,
	connMgr *engine.ConnManager) (da *DNSAgent) {
	da = &DNSAgent{cgrCfg: cgrCfg, fltrS: fltrS, connMgr: connMgr}
	return
}

// DNSAgent translates DNS requests towards CGRateS infrastructure
type DNSAgent struct {
	sync.RWMutex                   // used on services to lock/unlock DNSAgent
	cgrCfg       *config.CGRConfig // loaded CGRateS configuration
	fltrS        *engine.FilterS   // connection towards FilterS
	LASisON      bool              // Holds state of ListenAndServe function
	connMgr      *engine.ConnManager
}

type DNSAgentListener struct {
	server        *dns.Server   // server holds the DNS server configuration.
	lasWaitDefer  chan struct{} // signals the completion of DNS listener shutdown.
	unlockLASChan chan struct{} // used to synchronize and start DNS listening.
	errChan       chan error    // used to communicate errors encountered during DNS listening/creating TLS server.
}

// newDNSAgentListener will open a DNS Listener
func (dal *DNSAgentListener) newDNSAgentListener() {
	defer func() {
		dal.lasWaitDefer <- struct{}{} // signal 1 listener closing
	}()
	<-dal.unlockLASChan //continue only when no errors with TLS
	utils.Logger.Info(fmt.Sprintf("<%s> start listening on <%s:%s>",
		utils.DNSAgent, dal.server.Net, dal.server.Addr))
	err := dal.server.ListenAndServe()
	if err != nil {
		utils.Logger.Warning(fmt.Sprintf("<%s> error <%v>, on ListenAndServe <%s:%s>",
			utils.DNSAgent, err, dal.server.Net, dal.server.Addr))
		if strings.Contains(err.Error(), "address already in use") {
			return
		}
		dal.errChan <- err
	}
}

// ListenAndServe will run the DNS handler doing also the connection to listen address
func (da *DNSAgent) ListenAndServe(stopChan chan struct{}, shdComplete chan struct{}) (err error) {
	da.LASisON = true
	defer func() {
		da.LASisON = false
	}()
	errChan := make(chan error, 1)
	doneChan := make(chan struct{}) // used to indicate successfull shutdown of all DNS listeners
	unlockLASChan := make(chan struct{})
	lasWaitDefer := make(chan struct{}, len(da.cgrCfg.DNSAgentCfg().Listeners))
	go func() { // waiting to close function when all listeners are shut successfully
		<-lasWaitDefer
		close(doneChan)
	}()
	for i := range da.cgrCfg.DNSAgentCfg().Listeners {
		server := &dns.Server{
			Addr: da.cgrCfg.DNSAgentCfg().Listeners[i].Address,
			Net:  da.cgrCfg.DNSAgentCfg().Listeners[i].Network,
			Handler: dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
				go da.handleMessage(w, m)
			}),
		}
		if strings.HasSuffix(da.cgrCfg.DNSAgentCfg().Listeners[i].Network, utils.TLSNoCaps) {
			cert, err := tls.LoadX509KeyPair(da.cgrCfg.TLSCfg().ServerCerificate, da.cgrCfg.TLSCfg().ServerKey)
			if err != nil {
				err = fmt.Errorf("load certificate error <%v>", err)
				errChan <- err
				break
			}

			server.Net = "tcp-tls"
			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
		if i == len(da.cgrCfg.DNSAgentCfg().Listeners)-1 {
			close(unlockLASChan) // on last iteration signal waiting unlockLASChan to start listening on all servers
		}
		dnsAL := &DNSAgentListener{
			server:        server,
			lasWaitDefer:  lasWaitDefer,
			unlockLASChan: unlockLASChan,
			errChan:       errChan,
		}

		go dnsAL.newDNSAgentListener()

		go func() {
			<-stopChan // wait for stopChan signal to shut all alive DNS listeners
			utils.Logger.Info(fmt.Sprintf("<%s> Shutting down <%s:%s>",
				utils.DNSAgent, server.Net, server.Addr))
			err := server.Shutdown()
			if err != nil {
				utils.Logger.Warning(fmt.Sprintf("<%s> error <%v>, on Shutdown <%s:%s>",
					utils.DNSAgent, err, server.Net, server.Addr))
			}
		}()
	}
	select {
	case err := <-errChan:
		close(stopChan) // start shutting all dns servers at the same time
		return err
	case <-doneChan:
		close(shdComplete) // signal when all are shut without errors
		return nil
	}
}

// handleMessage is the entry point of all DNS requests
// requests are reaching here asynchronously
func (da *DNSAgent) handleMessage(w dns.ResponseWriter, req *dns.Msg) {
	dnsDP := newDnsDP(req)

	rply := newDnsReply(req)
	rmtAddr := w.RemoteAddr().String()
	for _, q := range req.Question {
		if processed, err := da.handleQuestion(dnsDP, rply, &q, rmtAddr); err != nil ||
			!processed {
			rply := newDnsReply(req)
			rply.Rcode = dns.RcodeServerFailure
			dnsWriteMsg(w, rply)
			return
		}
	}

	if err := dnsWriteMsg(w, rply); err != nil { // failed sending, most probably content issue
		rply := newDnsReply(req)
		rply.Rcode = dns.RcodeServerFailure
		dnsWriteMsg(w, rply)
	}
}

// handleMessage is the entry point of all DNS requests
// requests are reaching here asynchronously
func (da *DNSAgent) handleQuestion(dnsDP utils.DataProvider, rply *dns.Msg, q *dns.Question, rmtAddr string) (processed bool, err error) {
	reqVars := &utils.DataNode{
		Type: utils.NMMapType,
		Map: map[string]*utils.DataNode{
			utils.DNSQueryType: utils.NewLeafNode(dns.TypeToString[q.Qtype]),
			utils.DNSQueryName: utils.NewLeafNode(q.Name),
			utils.RemoteHost:   utils.NewLeafNode(rmtAddr),
		},
	}
	// message preprocesing
	cgrRplyNM := &utils.DataNode{Type: utils.NMMapType, Map: make(map[string]*utils.DataNode)}
	rplyNM := utils.NewOrderedNavigableMap() // share it among different processors
	opts := utils.MapStorage{}
	for _, reqProcessor := range da.cgrCfg.DNSAgentCfg().RequestProcessors {
		var lclProcessed bool
		if lclProcessed, err = processRequest(
			context.TODO(),
			reqProcessor,
			NewAgentRequest(
				dnsDP, reqVars, cgrRplyNM, rplyNM,
				opts, reqProcessor.Tenant,
				da.cgrCfg.GeneralCfg().DefaultTenant,
				utils.FirstNonEmpty(
					da.cgrCfg.DNSAgentCfg().Timezone,
					da.cgrCfg.GeneralCfg().DefaultTimezone,
				),
				da.fltrS, nil),
			utils.DNSAgent, da.connMgr,
			da.cgrCfg.DNSAgentCfg().SessionSConns,
			da.fltrS); err != nil {
			utils.Logger.Warning(
				fmt.Sprintf("<%s> error: %s processing message: %s from %s",
					utils.DNSAgent, err.Error(), dnsDP, rmtAddr))
			return
		}
		processed = processed || lclProcessed
		if lclProcessed && !reqProcessor.Flags.GetBool(utils.MetaContinue) {
			break
		}
	}
	if !processed {
		utils.Logger.Warning(
			fmt.Sprintf("<%s> no request processor enabled, ignoring message %s from %s",
				utils.DNSAgent, dnsDP, rmtAddr))
		return
	}
	if err = updateDNSMsgFromNM(rply, rplyNM, q.Qtype, q.Name); err != nil {
		utils.Logger.Warning(
			fmt.Sprintf("<%s> error: %s updating answer: %s from NM %s",
				utils.DNSAgent, err.Error(), utils.ToJSON(rply), utils.ToJSON(rplyNM)))
	}
	return
}
