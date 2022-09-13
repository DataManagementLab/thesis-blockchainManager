package deployer

import "BlockchainEnabler/BlockchainEnabler/internal/types"

type IDeployer interface {
	GenerateFiles(*types.Network, string,bool,bool) error
	Deploy(string) error
	Terminate(string) error
	// Monitor()x
	// Deploy()
	// GetServiceDefinition(string)
	// Log()
	// Remove()
	// Orchaestrate()
}
