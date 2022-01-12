package enablerplatform

import (
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain"
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain/fabric"
	"BlockchainEnabler/BlockchainEnabler/internal/conf"
	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

type EnablerPlatformManager struct {
	UserId   string
	Enablers []*types.EnablerPlatform
	Options  *conf.InitializationOptions
	logger   *zerolog.Logger
}

var EnablerPlatformCounter map[string]int

// initialization of the Enabler platform manager
var enablerManager *EnablerPlatformManager

func GetInstance(logger *zerolog.Logger) *EnablerPlatformManager {
	if enablerManager == nil {
		enablerManager = &EnablerPlatformManager{logger: logger}
	}
	return enablerManager
}

func (em *EnablerPlatformManager) InitEnablerPlatform(userId string, numberOfMembers int, initOptions *conf.InitializationOptions) (err error) {

	em.UserId = userId
	var e = new(types.EnablerPlatform)
	e.BlockchainProvider = initOptions.BlockchainType.String()
	e.ExposedBlockchainPort = initOptions.ServicesPort
	e.Members = make([]*types.Member, numberOfMembers)
	e.EnablerName = fmt.Sprintf("enabler_%s_%d", e.BlockchainProvider, em.GetCurrentCount(e.BlockchainProvider))
	em.logger.Printf("Initializing the members for the enabler")
	// Create members for each of the enabler ->
	// This members will be the different components that are needed and connected with the core.

	for i := 0; i < numberOfMembers; i++ {
		// externalProcess := i < options.ExternalProcesses
		e.Members[i] = createMember(fmt.Sprint(i), i, initOptions)
	}
	em.Enablers = append([]*types.EnablerPlatform{}, e)

	// Fetching the blockchain Provider
	//  setting the blockchain Provider..
	// Need to call a function which takes the e.BlockchainProvider and returns an Interface for the IProvider.-> which would be the fabric struct instance.

	e.InterfaceProvider = em.getBlockchainProvider(e)

	// now we need to provide the values that are needed to create our docker compose

	compose := docker.CreateDockerCompose()

	// Now need to call the service definition genrator.

	serviceDefinition := e.InterfaceProvider.GetDockerServiceDefinitions()
	for _, services := range serviceDefinition {
		compose.Services[services.ServiceName] = services.Service
		for _, volumeName := range services.VolumeNames {
			compose.Volumes[volumeName] = struct{}{}
		}
	}
	if err := em.ensureDirectories(e); err != nil {
		return err
	}
	if err := em.writeDockerCompose(compose, e); err != nil {
		return err
	}

	return nil
}

func (s *EnablerPlatformManager) writeDockerCompose(compose *docker.DockerComposeConfig, enabler *types.EnablerPlatform) error {
	bytes, err := yaml.Marshal(compose)
	if err != nil {
		return err
	}

	enablerDir := filepath.Join(constants.EnablerDir, s.UserId, enabler.EnablerName)

	return ioutil.WriteFile(filepath.Join(enablerDir, "docker-compose.yml"), bytes, 0755)
}

func (em *EnablerPlatformManager) ensureDirectories(s *types.EnablerPlatform) error {
	em.logger.Printf("The value for the userid %s", em.UserId)
	enablerDir := filepath.Join(constants.EnablerDir, em.UserId, s.EnablerName)

	if err := os.MkdirAll(filepath.Join(enablerDir, "configs"), 0755); err != nil {
		return err
	}

	for _, member := range s.Members {

		if err := os.MkdirAll(filepath.Join(enablerDir, "blockchain", member.ID), 0755); err != nil {
			return err
		}
	}
	return nil
}

func createMember(id string, index int, options *conf.InitializationOptions) *types.Member {
	serviceBase := options.ServicesPort + (index * 100)
	return &types.Member{
		ID:               id,
		Index:            &index,
		ExposedPort:      options.ServicesPort + index,
		ExposedAdminPort: serviceBase + 1, // note shared blockchain node is on zero
		OrgName:          options.OrgNames[index],
		NodeName:         options.NodeNames[index],
	}

}

func (em *EnablerPlatformManager) GetCurrentCount(s string) int {
	if len(EnablerPlatformCounter) == 0 {
		EnablerPlatformCounter = make(map[string]int)
		EnablerPlatformCounter[s] = 0
		return EnablerPlatformCounter[s]
	} else {
		if val, ok := EnablerPlatformCounter[s]; ok {
			return val + 1
		} else {
			EnablerPlatformCounter[s] = 0
			return EnablerPlatformCounter[s]
		}

	}
}

func (e *EnablerPlatformManager) getBlockchainProvider(enabler *types.EnablerPlatform) blockchain.IProvider {
	switch enabler.BlockchainProvider {
	case types.HyperledgerFabric.String():
		return fabric.GetFabricInstance(e.logger, enabler, "docker")
	default:
		return nil
	}
}
