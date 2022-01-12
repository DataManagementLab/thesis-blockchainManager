package fabric

import (
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain"
	"BlockchainEnabler/BlockchainEnabler/internal/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"

	// "BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"

	"github.com/rs/zerolog"
)

type FabricDefinition struct {
	Logger       *zerolog.Logger
	Enabler      *types.EnablerPlatform
	DeployerType string
	// Deployer     blockchain.IDeployer
}

var fab *FabricDefinition

func (f *FabricDefinition) Init() {
	// Need to call the deployer-> which can be anything from kubernetes to docker -> depending on the user choice.
	// by default it is docker
	// getDeployerInstance("docker")
	// running the docker init
	// getDeployerInstance(f.DeployerType).GenerateFiles(f.Enabler.EnablerName)
}

func GetFabricInstance(logger *zerolog.Logger, enabler *types.EnablerPlatform, deployerType string) *FabricDefinition {
	return &FabricDefinition{
		Logger:       logger,
		Enabler:      enabler,
		DeployerType: deployerType,
		// Deployer:     getDeployerInstance(deployerType),
	}
}
func (f *FabricDefinition) WriteConfigs() {

}

func getDeployerInstance(deployerType string) (deployer blockchain.IDeployer) {
	if deployerType == "docker" {
		return GetFabricDockerInstance()
	}
	return GetFabricDockerInstance()
}

func (f *FabricDefinition) GetDockerServiceDefinitions() []*docker.ServiceDefinition {
	f.Logger.Print("Fabric Service Definition function called.")
	return GenerateServiceDefinitions(f.Enabler.EnablerName)
}
