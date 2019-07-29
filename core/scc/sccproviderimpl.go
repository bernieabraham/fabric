/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scc

import (
	"fmt"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/pkg/errors"
)

// Provider implements sysccprovider.SystemChaincodeProvider
type Provider struct {
	Peer      *peer.Peer
	SysCCs    []SelfDescribingSysCC
	Whitelist Whitelist
}

// RegisterSysCC registers a system chaincode with the syscc provider.
func (p *Provider) RegisterSysCC(scc SelfDescribingSysCC) error {
	for _, registeredSCC := range p.SysCCs {
		if scc.Name() == registeredSCC.Name() {
			return errors.Errorf("chaincode with name '%s' already registered", scc.Name())
		}
	}
	p.SysCCs = append(p.SysCCs, scc)
	return nil
}

// IsSysCC returns true if the supplied chaincode is a system chaincode
func (p *Provider) IsSysCC(name string) bool {
	for _, sysCC := range p.SysCCs {
		if sysCC.Name() == name {
			return true
		}
	}
	if isDeprecatedSysCC(name) {
		return true
	}
	return false
}

// GetQueryExecutorForLedger returns a query executor for the specified channel
func (p *Provider) GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error) {
	l := p.Peer.GetLedger(cid)
	if l == nil {
		return nil, fmt.Errorf("Could not retrieve ledger for channel %s", cid)
	}

	return l.NewQueryExecutor()
}

// GetApplicationConfig returns the configtxapplication.SharedConfig for the channel
// and whether the Application config exists
func (p *Provider) GetApplicationConfig(cid string) (channelconfig.Application, bool) {
	return p.Peer.GetApplicationConfig(cid)
}

// Returns the policy manager associated to the passed channel
// and whether the policy manager exists
func (p *Provider) PolicyManager(channelID string) (policies.Manager, bool) {
	m := p.Peer.GetPolicyManager(channelID)
	return m, (m != nil)
}

func isDeprecatedSysCC(name string) bool {
	return name == "vscc" || name == "escc"
}

func (p *Provider) isWhitelisted(syscc SelfDescribingSysCC) bool {
	enabled, ok := p.Whitelist[syscc.Name()]
	return ok && enabled
}
