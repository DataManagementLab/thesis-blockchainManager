package deployer

import "BlockchainEnabler/BlockchainEnabler/internal/types"

type IDeployer interface {
	GenerateFiles(*types.EnablerPlatform,string) error
	// Monitor()x
	// GetServiceDefinition(interface{}) []*docker.ServiceDefinition
	// GenerateFiles(name string) interface{}
	// Deploy()
	// GetServiceDefinition(string)
	// Log()
	// Remove()
	// Orchaestrate()
}
