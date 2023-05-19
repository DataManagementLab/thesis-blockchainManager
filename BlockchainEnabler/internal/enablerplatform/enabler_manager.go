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
	"strings"
	"syscall"

	"github.com/rs/zerolog"
)

type EnablerPlatformManager struct {
	UserId   string
	Enablers []*types.Network
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
func (em *EnablerPlatformManager) InitEnablerPlatform(userId string, numberOfMembers int, initOptions *conf.InitializationOptions, localSetup bool) (err error) {

	em.UserId = userId
	var e = new(types.Network)
	e.BlockchainProvider = initOptions.BlockchainType.String()
	e.ExposedBlockchainPort = initOptions.ServicesPort
	e.Members = make([]*types.Member, numberOfMembers)
	e.NetworkName = initOptions.NetworkName
	em.logger.Printf("Initializing the members for the Network")
	// Create members for each of the network ->
	// This members will be the different components that are needed and connected with the core.

	for i := 0; i < numberOfMembers; i++ {
		e.Members[i] = em.createMember(fmt.Sprint(i), i, initOptions)
	}
	em.Enablers = append([]*types.Network{}, e)

	// now we need to provide the values that are needed to create our docker compose
	//
	// Here we can actually check which deployer is used and then call the functions related to that deployer.
	// There are currently two deployers to choose from 1. docker 2. K8
	// if the user chooses the docker deployment -> then the function needs to call the provider and then run the functions specific to the docker.
	// Otherwise it should call the functions specific to the k8s.

	// Creating the  structure.
	if err := em.ensureDirectories(e); err != nil {
		return err
	}
	// Fetching the blockchain Provider
	//  setting the blockchain Provider..
	// Need to call a function which takes the e.BlockchainProvider and returns an Interface for the IProvider.-> which would be the fabric struct instance.
	e.InterfaceProvider = em.getBlockchainProvider(e)
	if err := em.writeNetworkConfig(e); err != nil {
		return err
	}

	//  create a function which checks the ports and pass this function to the init.
	if err := e.InterfaceProvider.Init(em.UserId, initOptions.UseVolume, initOptions.BasicSetup, localSetup, initOptions.UserLogging); err != nil {
		return err
	}
	if err := em.writePlatformInfo(e); err != nil {
		return err
	}

	return nil
}

func (em *EnablerPlatformManager) CreateNetwork(useVolume bool, userLogging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			if err := network.InterfaceProvider.Create(em.UserId, false, useVolume, userLogging); err != nil {
				return err
			}
		}
	}
	// Things to do here
	// 0. checking if the ports are available or not and then starting the network
	// 1. calling the function for the blockchain network create.
	return nil
}

func (em *EnablerPlatformManager) CreateNetworkUsingSDK(useVolume bool, userLogging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			if err := network.InterfaceProvider.Create(em.UserId, true, false, userLogging); err != nil {
				return err
			}
		}
	}
	return nil
}

func (em *EnablerPlatformManager) writeNetworkConfig(enabler *types.Network) (err error) {
	orgDefinition := types.OrganizationDefinition{
		OrganizationName: enabler.Members[0].OrgName,
		ChannelName:      enabler.Members[0].ChannelName,
		OrdererName:      enabler.Members[0].OrdererName,
	}
	networkMembers := append([]*string{}, &enabler.Members[0].OrgName)

	fabricDefinition := types.FabricDefinition{
		BlockchainType:   enabler.BlockchainProvider,
		OrganizationInfo: orgDefinition,
		NetworkMembers:   networkMembers,
	}

	networkConfig := types.NetworkConfig{
		NetworkName:          enabler.NetworkName,
		BlockchainDefinition: fabricDefinition,
	}

	platformConfigBytes, err := json.MarshalIndent(networkConfig, "", " ")
	if err != nil {
		fmt.Println(err)
	}
	if err := ioutil.WriteFile(filepath.Join(constants.EnablerDir, em.UserId, networkConfig.NetworkName, "enabler", fmt.Sprintf("network_config.json")), platformConfigBytes, 0755); err != nil {
		return err
	}
	return nil
}

func (em *EnablerPlatformManager) writePlatformInfo(enabler *types.Network) (err error) {

	network := types.Network{
		NetworkName:           enabler.NetworkName,
		BlockchainProvider:    enabler.BlockchainProvider,
		ExposedBlockchainPort: enabler.ExposedBlockchainPort,
		Members:               enabler.Members,
	}
	platformConfigBytes, err := json.MarshalIndent(network, "", " ")
	if err != nil {
		fmt.Println(err)
	}
	if err := ioutil.WriteFile(filepath.Join(constants.EnablerDir, em.UserId, network.NetworkName, fmt.Sprintf("%s_info.json", network.NetworkName)), platformConfigBytes, 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(constants.EnablerDir, em.UserId, fmt.Sprintf("network_info.json")), platformConfigBytes, 0755); err != nil {
		return err
	}
	return nil
}
func (em *EnablerPlatformManager) LoadUser(netId string, userId string) error {
	var infoFile string
	if netId != "" {
		infoFile = filepath.Join(constants.EnablerDir, userId, netId, fmt.Sprintf("%s_info.json", netId))
	} else {
		infoFile = filepath.Join(constants.EnablerDir, userId, fmt.Sprintf("network_info.json"))
	}
	// can read from the json file outside the names of the networks that are created and then looping through them and opening them.
	// or can use a file which is outside which contains all the info to the different networks and is appended one thing this would do is making things easier while searching for port used.
	em.logger.Printf("Loading the Network ....")
	em.logger.Printf("location for the create command %s", infoFile)
	var network *types.Network
	read, err := ioutil.ReadFile(infoFile)
	if err != nil {
		return err
	}
	json.Unmarshal(read, &network)
	network.InterfaceProvider = em.getBlockchainProvider(network)

	em.Enablers = append(em.Enablers, network)
	// check for which provider it belongs to.
	em.logger.Printf("Network loaded successfully.")
	em.UserId = userId
	return nil
}

func (em *EnablerPlatformManager) ensureDirectories(s *types.Network) error {
	enablerDir := filepath.Join(constants.EnablerDir, em.UserId, s.NetworkName)

	syscall.Umask(0)
	if err := os.MkdirAll(filepath.Join(enablerDir, "configs"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(enablerDir, "enabler"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(enablerDir, "enabler", "chaincode"), 0777); err != nil {
		return err
	}

	for _, member := range s.Members {

		if err := os.MkdirAll(filepath.Join(enablerDir, "blockchain", member.ID), 0777); err != nil {
			return err
		}
	}
	return nil
}

func (em *EnablerPlatformManager) AddOrganization(useVolume bool, file string,logging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			return network.InterfaceProvider.Add(em.UserId, useVolume, file,logging)
		}
	}
	return nil
}

func (em *EnablerPlatformManager) SignOrganization(useVolume bool, file string, update bool, logging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			return network.InterfaceProvider.Sign(em.UserId, useVolume, file, update,logging)
		}
	}
	return nil
}

func (em *EnablerPlatformManager) DeleteNetwork(logging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			return network.InterfaceProvider.Delete(em.UserId,logging)
		}
	}
	return nil
}

func (em *EnablerPlatformManager) JoinNetwork(useVolume bool, zipFile string, basicSetup bool,logging bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			return network.InterfaceProvider.Join(em.UserId, useVolume, zipFile, basicSetup,logging)
		}
	}
	return nil
}

func (em *EnablerPlatformManager) LeaveNetwork(networkId string, orgName string, useVolume bool, finalize bool) error {
	if em.Enablers != nil {
		for _, network := range em.Enablers {
			return network.InterfaceProvider.Leave(networkId, orgName, em.UserId, useVolume, finalize)
		}
	}
	return nil
}

func (em *EnablerPlatformManager) createMember(id string, index int, options *conf.InitializationOptions) *types.Member {
	if options.ServicesPort == 0 {
		options.ServicesPort = 5000
	}
	serviceBase := options.ServicesPort + (index * 100)
	return &types.Member{
		ID:               id,
		Index:            &index,
		ExposedPort:      options.ServicesPort + index,
		ExposedAdminPort: serviceBase + 1, // note shared blockchain node is on zero
		OrgName:          fmt.Sprintf("%s", options.OrgNames[index]),
		NodeName:         fmt.Sprintf("%s", options.NodeNames[index]),
		OrdererOrg:       fmt.Sprintf("Orderer%s", options.OrgNames[index]),
		OrdererName:      fmt.Sprintf("fabric_orderer.%s", strings.ToLower(options.OrgNames[index])),
		DomainName:       "example.com",
		ChannelName:      fmt.Sprintf("channel%s", strings.ToLower(options.OrgNames[index])),
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

func (e *EnablerPlatformManager) getBlockchainProvider(network *types.Network) blockchain.IProvider {
	switch network.BlockchainProvider {
	case types.HyperledgerFabric.String():
		return fabric.GetFabricInstance(e.logger, network, "docker")
	default:
		return nil
	}
}
