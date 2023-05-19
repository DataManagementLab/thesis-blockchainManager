package deployer

import "BlockchainEnabler/BlockchainEnabler/internal/types"

type IDeployer interface {
	GenerateFiles(*types.Network, string, bool, bool) error
	Deploy(string, bool) error
	Terminate(string, bool) error
	// Monitor()x
	// Deploy()
	// GetServiceDefinition(string)
	// Log()
	// Remove()
	// Orchaestrate()
}
