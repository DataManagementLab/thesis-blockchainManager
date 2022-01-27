package types

import (
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain"
	// "BlockchainEnabler/BlockchainEnabler/internal/blockchain/fabric"
)

type EnablerPlatform struct {
	EnablerName           string    `json:"name,omitempty"`
	Members               []*Member `json:"members,omitempty"`
	ExposedBlockchainPort int       `json:"exposedPort,omitempty"`
	BlockchainProvider    string    `json:"blockchainProvider"`
	InterfaceProvider     blockchain.IProvider
	// InterfaceDeployer blockchain.IDeployer
}

type Member struct {
	ID               string      `json:"id,omitempty"`
	Index            *int        `json:"index,omitempty"`
	Address          string      `json:"address,omitempty"`
	ExposedPort      int         `json:"exposedPort"`
	ExposedAdminPort int         `json:"exposedAdminPort,omitempty"`
	ExternalPorts    interface{} `json:"externalPorts,omitempty"`
	OrgName          string      `json:"orgName,omitempty"`
	NodeName         string      `json:"nodeName,omitempty"`
}
