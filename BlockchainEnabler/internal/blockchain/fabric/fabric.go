package fabric

import (
	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	_ "embed"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"time"

	"BlockchainEnabler/BlockchainEnabler/internal/deployer"

	// "BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"

	"github.com/rs/zerolog"
)

type FabricDefinition struct {
	Logger       *zerolog.Logger
	Enabler      *types.Network
	DeployerType string
	Deployer     deployer.IDeployer
}

var fab *FabricDefinition

//go:embed configtx.yaml
var configtxYaml string
var userIdentification string

func (f *FabricDefinition) Init(userId string) (err error) {

	//Steps to follow:
	// Basic step to fetch the deployer instance.
	// call the deployer init function then -> deployer init will create the dockercompose basic setup.
	// 1.Creating docker compose
	// 2. ensure directories
	// 3. write configs
	// 4. write docker compose

	// Current decision is to take the docker as default deployment platform.

	// check if the fabric deployertype is docker then initialze deployer with it.
	f.Deployer = getDeployerInstance(f.DeployerType)
	userIdentification = userId
	// once this is done then need to call the deployer init.
	// call the deployer file generation.
	f.setValidPorts()
	if err := f.Deployer.GenerateFiles(f.Enabler, userId); err != nil {
		return err
	}
	if err := f.writeConfigs(userId); err != nil {
		return err
	}
	// Need to call the deployer-> which can be anything from kubernetes to docker -> depending on the user choice.
	// by default it is docker
	// getDeployerInstance("docker")
	// running the docker init
	// getDeployerInstance(f.DeployerType).GenerateFiles(f.Enabler.NetworkName)
	return nil
}

func GetFabricInstance(logger *zerolog.Logger, enabler *types.Network, deployerType string) *FabricDefinition {
	return &FabricDefinition{
		Logger:       logger,
		Enabler:      enabler,
		DeployerType: deployerType,
		// Deployer:     getDeployerInstance(deployerType),
	}
}

func writeConfigtxYaml(blockchainPath string) error {

	filePath := path.Join(blockchainPath, "configtx.yaml")
	return ioutil.WriteFile(filePath, []byte(configtxYaml), 0755)
}

// The port checker functionality can be implemented in the enabler_manager and then it is passed as a function here too, as the fabric would have an implementation for
// the ports it wishes to utilize.
func (f *FabricDefinition) writeConfigs(userId string) (err error) {

	// Steps to be handled here
	// 1. Create cryptogen config file
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	// Need to also check if the ports are available or not. If the ports are not available then just use a different port

	// Call to method check ports and then assigning the ports for the docker compose and others.

	// also need to add certain ports and check for certain ports from the member.

	if err := WriteCryptogenConfig(1, cryptogenYamlPath); err != nil {
		return err
	}

	if err := WriteNetworkConfig(path.Join(blockchainDirectory, "ccp.yaml")); err != nil {
		return err
	}
	if err := writeConfigtxYaml(blockchainDirectory); err != nil {
		return err
	}

	// 2.  Create the Network config file alias ccp.yaml file

	//  3. Can create the fabconnect file -> fabconnect.yaml

	//  4. Create the configtx.yaml file.

	// Also specify why each of these files are create here.
	// For example when creating the members with different nodes can actually decide
	//  on how many orderers are needed in the file as well as the number of peers and organizations.
	// Note: This part can be skipped now as we have currently decided on using only 1 peer, 1 org, 1orderer,1 ca when starting the org

	// Here we need to create the cryptogen config file

	return nil
}
func (f *FabricDefinition) setValidPorts() {
	// Assign the member with the external ports that are required for the certificate authority, orderer and peers.
	host := "127.0.0.1"
	for _, member := range f.Enabler.Members {
		if res := checkPortIsOpened(host, member.ExposedPort); res == false {
			for i := 1; i < 5; i++ {
				if checkPortIsOpened(host, member.ExposedPort+i*100) {
					// set the exposed admin port to then other values.
					member.ExposedAdminPort = member.ExposedPort + i*100 + 1
					member.ExposedPort = member.ExposedPort + i*100
					break
				}
			}

		}
		member.ExternalPorts = setExternalPorts(member)
		for _, port := range member.ExternalPorts.(map[string]int) {
			if res := checkPortIsOpened(host, port); res == false {
				for i := 1; i < 5; i++ {
					if checkPortIsOpened(host, port+i*100) {
						// set the exposed admin port to then other values.
						port = port + i*100
						break
					}
				}

			}
		}
		// first check the basic ports if they are available or not.
		// now once we have the member need to check the port available and then set it accordingly
	}
}
func setExternalPorts(mem *types.Member) map[string]int {
	external := map[string]int{
		"ca_server_port":                       7054,
		"ca_operations_listen_port":            17054,
		"orderer_general_listen_port":          7050,
		"orderer_admin_listen_port":            7053,
		"orderer_operations_listen_port":       17050,
		"core_peer_listen_address_gossip_port": 7051,
		"core_peer_chaincode_listen_port":      7052,
		"core_operations_listen_port":          17051,
	}
	return external

}

func getDeployerInstance(deployerType string) (deployer deployer.IDeployer) {
	if deployerType == "docker" {
		return GetFabricDockerInstance()
	}
	return GetFabricDockerInstance()
}

func (f *FabricDefinition) Create() (err error) {
	f.generateGenesisBlock(userIdentification)
	// Step to do inside the create function

	// 1.Also need to check if the docker is present in the host machine.
	// 2. We would need to run the first time setup where the initiailization of blockcahin node happens.

	// Currently i am planning to use the functions the docker code from the firefly cli seems quite nice way of handling things.
	return nil
}

// setting up the docker container and the volume and running the cryptogen configs

func (f *FabricDefinition) generateGenesisBlock(userId string) (err error) {
	verbose := true
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	volumeName := fmt.Sprintf("%s_enabler_fabric", f.Enabler.NetworkName)

	if err := docker.CreateVolume(volumeName, verbose); err != nil {
		return err
	}

	// Run cryptogen to generate MSP
	if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
		return err
	}

	// Generate genesis block
	if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", "enablerchannel"); err != nil {
		return err
	}

	return nil
}

// We could actually check out with the deployer instance if it is docker then using the getDockerServiceDefinition,
// other using something similar for the K8s.

// Now another thing is we are assigning the GetDockerService Definition in the interface,
// which we could actually avoid and instead genrate and call the methods related to the generation of the docker compose file
// inside here instead of in the enabler_manager.

// so once that is done we would be using the fabric - docker instance to call the docker methods.
// The fabric will implement the methods that it needs as init, create,join and leave
// but inside those it will call the specific deployer instance functions.
// func (f *FabricDefinition) GetDockerServiceDefinitions() []*docker.ServiceDefinition {
// 	f.Logger.Print("Fabric Service Definition function called.")
// 	return GenerateServiceDefinitions(f.Enabler)
// }

func checkPortIsOpened(host string, port int) bool {
	timeout := time.Millisecond * 500
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

// Few things need to be changed

// 1. the structure and the interfaces need to be changed a bit
// 2. Not so sure about the applicability of the enabler as well as the manager dont really know if it would be necessary

// 2.1 The main idea behind the enabler manager is to run multiple enablers at the same time
// 2.2 Also to manage the members(the port informations) for each of the enablers such that no new enabler is given the same port
// 2.3 The things most important to work with now is to generate the config file and then loading the config file.
// 2.4 Once the config file is loaded then using the docker containers to run the setup.

// Current order of how things will be implemented
// Tuesday -> working on cleaning the code structure a bit + creating the config file (currently without worrying about ports)
// Wednesday -> if the config file is getting generated, the main task is to create the keys, and the other things needed to load the config
// 				which would be to check the firefly code for inspiration on how to generate the keys and then run the keys inside(/outside)
// 				if the keys are common then they need to be stored in our local host machine.
// Thursday -> working on the proposal a bit the idea is to have a basic create network done by this week along with the class diagram.

// The work seems a lot but can be definitely done just need to keep at it and avoid making the same mistakes i have made earlier.
// Things get better and easier you just have to do it every day.

// Also need to keep a track of how far i have progressed every day or every two days, keep a plan for that.

// I really need to finish the main stuff this week before the call as might open up a possiblity for job offer which i need to search now.
