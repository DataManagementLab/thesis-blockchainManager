package blockchain

import "BlockchainEnabler/BlockchainEnabler/internal/docker"

// "BlockchainEnabler/BlockchainEnabler/internal/docker"

// type IBlockchainEnabler interface{
// 	IDeployer
// 	IProvider
// }

type IDeployer interface {
	// Monitor()x
	// GetServiceDefinition(interface{}) []*docker.ServiceDefinition
	// GenerateFiles(name string) interface{}
	// Deploy()
	// GetServiceDefinition(string)
	// Log()
	// Remove()
	// Orchaestrate()
}

type IProvider interface {
	Init()
	WriteConfigs()
	GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
