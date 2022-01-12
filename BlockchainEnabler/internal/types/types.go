package types

import (
	"fmt"
	"strings"
)

var BlockchainProvidersList =[]string{"fabric","geth","corda"}
type BlockchainProvider int
const (
	HyperledgerFabric BlockchainProvider = iota
	Ethereum 
	Corda
)

func (b BlockchainProvider) String() string{
	return BlockchainProvidersList[b]
}

func BlockchainProviderSelection(s string) (BlockchainProvider,error){
	for i, blockchains := range BlockchainProvidersList{
		if strings.ToLower(s) == blockchains{
			return BlockchainProvider(i),nil
		}
	}
	return HyperledgerFabric, fmt.Errorf("\"%s\" is not a valid provider selection, the possible selections are %v",s,BlockchainProvidersList)
}