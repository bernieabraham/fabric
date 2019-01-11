/*
 *
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 * /
 *
 */

package endorser

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
)

// PvtRWSetAssembler assembles private read write set for distribution
// augments with additional information if needed
type PvtRWSetAssembler interface {
	// AssemblePvtRWSet prepares TxPvtReadWriteSet for distribution
	// augmenting it into TxPvtReadWriteSetWithConfigInfo adding
	// information about collections config available related
	// to private read-write set
	AssemblePvtRWSet(privData *rwset.TxPvtReadWriteSet,
		txsim ledger.SimpleQueryExecutor,
		deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider) (
		*transientstore.TxPvtReadWriteSetWithConfigInfo, error,
	)
}

// CollectionConfigRetriever encapsulates sub-functionality of ledger.TxSimulator
// to abstract minimum required functions set
type CollectionConfigRetriever interface {
	// GetState gets the value for given namespace and key. For a chaincode, the namespace corresponds to the chaincodeId
	GetState(namespace string, key string) ([]byte, error)
}

type rwSetAssembler struct {
}

// AssemblePvtRWSet prepares TxPvtReadWriteSet for distribution
// augmenting it into TxPvtReadWriteSetWithConfigInfo adding
// information about collections config available related
// to private read-write set
func (as *rwSetAssembler) AssemblePvtRWSet(privData *rwset.TxPvtReadWriteSet,
	txsim ledger.SimpleQueryExecutor, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider) (
	*transientstore.TxPvtReadWriteSetWithConfigInfo, error,
) {
	txPvtRwSetWithConfig := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset:          privData,
		CollectionConfigs: make(map[string]*common.CollectionConfigPackage),
	}

	for _, pvtRwset := range privData.NsPvtRwset {
		namespace := pvtRwset.Namespace
		if _, found := txPvtRwSetWithConfig.CollectionConfigs[namespace]; !found {
			ccInfo, err := deployedCCInfoProvider.ChaincodeInfo(namespace, txsim)
			if err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("error while retrieving collection config for chaincode %#v", namespace))
			}
			colCP := ccInfo.CollectionConfigPkg
			if colCP == nil {
				return nil, errors.New(fmt.Sprintf("no collection config for chaincode %#v", namespace))
			}
			txPvtRwSetWithConfig.CollectionConfigs[namespace] = colCP
		}
	}
	as.trimCollectionConfigs(txPvtRwSetWithConfig)
	return txPvtRwSetWithConfig, nil
}

func (as *rwSetAssembler) trimCollectionConfigs(pvtData *transientstore.TxPvtReadWriteSetWithConfigInfo) {
	flags := make(map[string]map[string]struct{})
	for _, pvtRWset := range pvtData.PvtRwset.NsPvtRwset {
		namespace := pvtRWset.Namespace
		for _, col := range pvtRWset.CollectionPvtRwset {
			if _, found := flags[namespace]; !found {
				flags[namespace] = make(map[string]struct{})
			}
			flags[namespace][col.CollectionName] = struct{}{}
		}
	}

	filteredConfigs := make(map[string]*common.CollectionConfigPackage)
	for namespace, configs := range pvtData.CollectionConfigs {
		filteredConfigs[namespace] = &common.CollectionConfigPackage{}
		for _, conf := range configs.Config {
			if colConf := conf.GetStaticCollectionConfig(); colConf != nil {
				if _, found := flags[namespace][colConf.Name]; found {
					filteredConfigs[namespace].Config = append(filteredConfigs[namespace].Config, conf)
				}
			}
		}
	}
	pvtData.CollectionConfigs = filteredConfigs
}
