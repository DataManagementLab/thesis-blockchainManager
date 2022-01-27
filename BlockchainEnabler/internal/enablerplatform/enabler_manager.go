package enablerplatform

import (
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain"
	"BlockchainEnabler/BlockchainEnabler/internal/blockchain/fabric"
	"BlockchainEnabler/BlockchainEnabler/internal/conf"
	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
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

// This function initializes the Enabler Platform.
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

	// now we need to provide the values that are needed to create our docker compose
	//
	// Here we can actually check which deployer is used and then call the functions related to that deployer.
	// There are currently two deployers to choose from 1. docker 2. K8
	// if the user chooses the docker deployment -> then the function needs to call the provider and then run the functions specific to the docker.
	// Otherwise it should call the functions specific to the k8s.
	//
	// Creating the directory structure.
	if err := em.ensureDirectories(e); err != nil {
		return err
	}
	if err := em.writePlatformInfo(e); err != nil {
		return err
	}
	e.InterfaceProvider = em.getBlockchainProvider(e)

	if err := e.InterfaceProvider.Init(em.UserId); err != nil {
		return err
	}

	return nil
}

func (em *EnablerPlatformManager) writePlatformInfo(enabler *types.EnablerPlatform) (err error) {
	platformConfigBytes, err := json.MarshalIndent(enabler, "", " ")
	if err != nil {
		fmt.Println(err)
	}
	if err := ioutil.WriteFile(filepath.Join(constants.EnablerDir, em.UserId, "platform_info.json"), platformConfigBytes, 0755); err != nil {
		return err
	}
	return nil
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
