package fabric

import (
	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	_ "embed"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"syscall"
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
var verbose bool

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
	verbose = true
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
	host := "0.0.0.0"
	for _, member := range f.Enabler.Members {
		if res, _ := checkPortIsOpened(host, member.ExposedPort); res == false {
			for i := 1; i < 5; i++ {
				if res, _ := checkPortIsOpened(host, member.ExposedPort+i*100); res {
					// set the exposed admin port to then other values.
					member.ExposedAdminPort = member.ExposedPort + i*100 + 1
					member.ExposedPort = member.ExposedPort + i*100
					break
				}
			}

		}
		member.ExternalPorts = setExternalPorts(member)
		for _, port := range member.ExternalPorts.(map[string]int) {
			if res, _ := checkPortIsOpened(host, port); res == false {
				for i := 1; i < 5; i++ {
					if res, _ := checkPortIsOpened(host, port+i*100); res {
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

func (f *FabricDefinition) Create(userId string) (err error) {
	// Step to do inside the create function

	// 1.Also need to check if the docker is present in the host machine.
	// 2. We would need to run the first time setup where the initiailization of blockcahin node happens.
	verbose = true
	f.Deployer = getDeployerInstance(f.DeployerType)
	userIdentification = userId
	f.generateGenesisBlock(userId)
	workingDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	fmt.Printf("Working directory %s", workingDir)

	if err := f.Deployer.Deploy(workingDir); err != nil {
		return err
	}

	f.createChannel(userId)
	f.joinChannel(userId)
	f.getBlockInformation(userId)
	// Next step is to actually run the container and pass the parameter in the containers.
	// For this particular use case we will get hte docker instance of the machine and then run the container in the fabric_docker file.
	// This container start up can be different according to the container so for example the startup function in the deployerinterface should be created.

	// Currently i am planning to use the functions the docker code from the firefly cli seems quite nice way of handling things.
	return nil
}

func (f *FabricDefinition) Join(networkId string, orgName string, networkId2 string, joiningOrgName string, userid string) (err error) {
	// Starting step would be to check if the network is already present and if so then it would kind of load the network.(Dont exactly know how it should load the network)
	// The first step can be to try to get the location for the org and then using the tool to generate the files needed.
	// the files which are needed are the crypto, configtx, docker-compose.
	// Currently we will just load these files and not create them.
	userIdentification = userid
	verbose = true
	f.Deployer = getDeployerInstance(f.DeployerType)
	// First checking if the network is already present, other wise creating the org3 structure
	f.createOrganizationForJoin(userid, networkId, orgName)
	// The previous step should be asynchronous though

	f.fetchConfigBlock(userid)

	f.envelopeBlockCreation(userid, networkId, orgName)
	// So go to the folder structure and then create a docker instance and then create a volume, copy files in the volume. -> crypto, configtx
	// Once volume is done then create the crypto files using crypto command.
	return nil
}

func (f *FabricDefinition) createOrganizationForJoin(userId string, networkId string, orgName string) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, networkId, "blockchain")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	configtxPath := path.Join(blockchainDirectory, "configtx.yaml")
	// volumeName := fmt.Sprintf("%s_fabric", networkId)
	enablerPath := path.Join(constants.EnablerDir, userId, networkId, "enabler")
	// f.Logger.Printf("Generating the volume with volume name: %s", volumeName)
	// if err := docker.CreateVolume(volumeName, verbose); err != nil {
	// 	return err
	// }
	f.Logger.Printf("Using the fabric tools to generate the msp with cryptogen tool in the shared volume location")
	// Run cryptogen to generate MSP
	if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/enabler/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
		return err
	}
	// Running the configtxgen command to get the org3 definition and store it in the current folder.
	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", configtxPath, enablerPath, orgName, enablerPath, orgName)).Output()

	if err != nil {
		return err
	}
	fmt.Printf(" %s\n", out)
	return nil
}

func (f *FabricDefinition) fetchConfigBlock(userId string) (err error) {
	f.Logger.Printf("Fetching config block for channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		"CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", "enablerchannel", "-o", "fabric_orderer:7050", "--tls", "--cafile", "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem")
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		"CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", f.Enabler.NetworkName, volumeName, networkDir)).Output()
	if err != nil {
		return err
	}
	fmt.Printf(" %s\n", out)
	return nil
}
func (f *FabricDefinition) envelopeBlockCreation(userId string, networkId string, orgName string) (err error) {
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	// Required Step

	enablerPath := path.Join(constants.EnablerDir, userId, networkId, "enabler")
	orgDefFilePath := path.Join(enablerPath, "Org3.json")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	// Required Step

	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s:/etc/enabler/org3.json -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq -s '.[0] * {\"channel_group\":{\"groups\":{\"Application\":{\"groups\": {\"Org3MSP\":.[1]}}}}}' /etc/enabler/config1.json /etc/enabler/org3.json > %s/enabler/modified_config.json ", f.Enabler.NetworkName, volumeName, orgDefFilePath, networkDir, networkDir)).Output()
	if err != nil {
		return err
	}
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", "enablerchannel", "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_config.pb", "--output", "/etc/enabler/org3_update.pb")
	// Required Step
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/org3_update.pb --type common.ConfigUpdate | jq . > %s/enabler/org3_update.json", f.Enabler.NetworkName, volumeName, networkDir)).Output()

	// Required Step

	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"enablerchannel\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/org3_update.json)'}}}'| jq . > %s/enabler/org3_update_in_envelope.json", f.Enabler.NetworkName, volumeName, networkDir, networkDir))

	fmt.Printf("%s", cmd.String())
	out, err = cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf(" Printng out %s\n", out)

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s/enabler/org3_update_in_envelope.json:/etc/enabler/org3_update_in_envelope.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/org3_update_in_envelope.json", "--type", "common.Envelope", "--output", "/etc/enabler/org3_update_in_envelope.pb")

	return nil
}

// setting up the docker container and the volume and running the cryptogen configs

func (f *FabricDefinition) generateGenesisBlock(userId string) (err error) {
	verbose := true
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)

	// volumeName := fmt.Sprintf("enabler_fabric")
	f.Logger.Printf("Generating the volume with volume name: %s", volumeName)
	if err := docker.CreateVolume(volumeName, verbose); err != nil {
		return err
	}
	f.Logger.Printf("Using the fabric tools to generate the msp with cryptogen tool in the shared volume location")
	// Run cryptogen to generate MSP
	if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
		return err
	}

	f.Logger.Printf("Using the fabric tools to generate the Gensis block in the shared volume location")
	// Generate genesis block
	// might also need to generate the configtx yaml file according the orgname and even the name as example.com does not seem quite good enough
	fmt.Printf("Location of the configtx file %s", path.Join(blockchainDirectory, "configtx.yaml"))
	if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", "enablerchannel"); err != nil {
		//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
		return err
	}

	return nil
}

func (f *FabricDefinition) createChannel(userId string) (err error) {
	verbose := true
	f.Logger.Printf("Creating channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "osnadmin", "channel", "join", "--channelID", "enablerchannel", "--config-block", "/etc/enabler/enabler.block", "-o", "fabric_orderer:7053", "--ca-file", "/etc/enabler/organizations/ordererOrganizations/example.com/users/Admin@example.com/tls/ca.crt", "--client-cert", "/etc/enabler/organizations/ordererOrganizations/example.com/users/Admin@example.com/tls/client.crt", "--client-key", "/etc/enabler/organizations/ordererOrganizations/example.com/users/Admin@example.com/tls/client.key")
}

func (f *FabricDefinition) joinChannel(userId string) error {
	verbose := true
	f.Logger.Printf("Joining channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e", "CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/enabler.block")
}

func (f *FabricDefinition) getBlockInformation(userId string) error {
	verbose := true
	f.Logger.Printf("Get block information")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "configtxlator", "proto_decode", "--input", "/etc/enabler/enabler.block", "--output", "/etc/enabler/enabler.json", "--type", "common.Block")

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

func checkPortIsOpened(host string, port int) (bool, error) {
	// timeout := time.Millisecond * 500

	timeout := time.Millisecond * 500
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)), timeout)

	if netError, ok := err.(net.Error); ok && netError.Timeout() {
		return true, nil
	}

	switch t := err.(type) {

	case *net.OpError:
		switch t := t.Unwrap().(type) {
		case *os.SyscallError:
			if t.Syscall == "connect" {
				return true, nil
			}
		}
		if t.Op == "dial" {
			return false, err
		} else if t.Op == "read" {
			return true, nil
		}

	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			return true, nil
		}
	}

	if conn != nil {
		defer conn.Close()
		return false, nil
	}
	return true, nil

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
