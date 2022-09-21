package types

import (
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain"
	// "BlockchainEnabler/BlockchainEnabler/internal/blockchain/fabric"
)

type Network struct {
	NetworkName           string               `json:"name,omitempty"`
	Members               []*Member            `json:"members,omitempty"`
	ExposedBlockchainPort int                  `json:"exposedPort,omitempty"`
	BlockchainProvider    string               `json:"blockchainProvider,omitempty"`
	InterfaceProvider     blockchain.IProvider `json:"provider,omitempty"`
	// InterfaceDeployer blockchain.IDeployer
}

type NetworkConfig struct {
	NetworkName          string           `json:"networkname,omitempty"`
	BlockchainDefinition FabricDefinition `json:"blockchain,omitempty"`
}

type FabricDefinition struct {
	BlockchainType   string                 `json:"type,omitempty"`
	OrganizationInfo OrganizationDefinition `json:"organization,omitempty"`
	NetworkMembers   []*string              `json:"organizationlist,omitempty"`
}

type OrganizationDefinition struct {
	OrganizationName string `json:"orgname,omitempty"`
	OrdererName      string `json:"orderername,omitempty"`
	ChannelName      string `json:"channelname,omitempty"`
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
	OrdererName      string      `json:"ordererName,omitempty"`
	OrdererOrg       string      `json:"ordererOrg,omitempty"`
	DomainName       string      `json:"domainName,omitempty"`
	ChannelName      string      `json:"channelName,omitempty"`
}
