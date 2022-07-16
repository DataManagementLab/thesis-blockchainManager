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
	"regexp"
	"strings"
	"syscall"
	"time"

	"BlockchainEnabler/BlockchainEnabler/internal/deployer"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"

	// "BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"

	"github.com/rs/zerolog"
)

type FabricDefinition struct {
	Logger       *zerolog.Logger
	Enabler      *types.Network
	DeployerType string
	Deployer     deployer.IDeployer
	UseVolume    bool
}

var fab *FabricDefinition

//go:embed configtx.yaml
var configtxYaml string

//go:embed configtx-basicsetup.yaml
var configtxBasicSetupYaml string
var userIdentification string
var verbose bool

func (f *FabricDefinition) Init(userId string, useVolume bool, basicSetup bool) (err error) {

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
	if err := f.Deployer.GenerateFiles(f.Enabler, userId, useVolume, basicSetup); err != nil {
		return err
	}
	if err := f.writeConfigs(userId, f.Enabler.Members[0], basicSetup); err != nil {
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

func writeConfigtxYaml(blockchainPath string, basicSetup bool) error {

	filePath := path.Join(blockchainPath, "configtx.yaml")
	if basicSetup {
		return ioutil.WriteFile(filePath, []byte(configtxBasicSetupYaml), 0755)
	}
	return ioutil.WriteFile(filePath, []byte(configtxYaml), 0755)
}

func transformConfigtxYaml(blockchainPath string, basicSetup bool, orgName string, ordererName string) error {
	filePath := path.Join(blockchainPath, "configtx.yaml")
	if basicSetup {
		re := regexp.MustCompile(`(Org3)`)
		replaced := re.ReplaceAllString(configtxBasicSetupYaml, fmt.Sprintf("%s", orgName))
		re = regexp.MustCompile("org3")
		replaced = re.ReplaceAllString(replaced, fmt.Sprintf("%s", strings.ToLower(orgName)))

		return ioutil.WriteFile(filePath, []byte(replaced), 0755)
	}
	re := regexp.MustCompile(`(Org1)`)
	replaced := re.ReplaceAllString(configtxYaml, fmt.Sprintf("%s", orgName))
	re = regexp.MustCompile("org1")
	replaced = re.ReplaceAllString(replaced, fmt.Sprintf("%s", strings.ToLower(orgName)))
	re = regexp.MustCompile("OrdererOrg")
	replaced = re.ReplaceAllString(replaced, fmt.Sprintf("Orderer%s", orgName))
	re = regexp.MustCompile("OrdererMSP")
	replaced = re.ReplaceAllString(replaced, fmt.Sprintf("Orderer%sMSP", orgName))
	re = regexp.MustCompile("fabric_orderer")
	replaced = re.ReplaceAllString(replaced, fmt.Sprintf("%s", ordererName))

	return ioutil.WriteFile(filePath, []byte(replaced), 0755)

	// return nil
}

// The port checker functionality can be implemented in the enabler_manager and then it is passed as a function here too, as the fabric would have an implementation for
// the ports it wishes to utilize.
func (f *FabricDefinition) writeConfigs(userId string, net *types.Member, basicSetup bool) (err error) {

	// Steps to be handled here
	// 1. Create cryptogen config file
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	// Need to also check if the ports are available or not. If the ports are not available then just use a different port

	// Call to method check ports and then assigning the ports for the docker compose and others.

	// also need to add certain ports and check for certain ports from the member.

	if err := WriteCryptogenConfig(1, cryptogenYamlPath, net, basicSetup); err != nil {
		return err
	}

	if err := WriteNetworkConfig(path.Join(blockchainDirectory, "ccp.yaml"), path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler"), *net); err != nil {
		return err
	}
	// if err := writeConfigtxYaml(blockchainDirectory, basicSetup); err != nil {
	// 	return err
	// }
	if err := transformConfigtxYaml(blockchainDirectory, basicSetup, net.OrgName, net.OrdererName); err != nil {
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

func (f *FabricDefinition) Create(userId string, useSDK bool, useVolume bool, basicSetup bool) (err error) {
	// Step to do inside the create function

	// 1.Also need to check if the docker is present in the host machine.
	// 2. We would need to run the first time setup where the initiailization of blockcahin node happens.
	verbose = true
	f.Deployer = getDeployerInstance(f.DeployerType)
	userIdentification = userId
	workingDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	f.generateCryptoMaterial(userId, useVolume)
	if basicSetup {
		if err := f.Deployer.Deploy(workingDir); err != nil {
			return err
		}
		return nil
	}
	f.generateGenesisBlock(userId, useVolume)

	fmt.Printf("Working directory %s", workingDir)

	if err := f.Deployer.Deploy(workingDir); err != nil {
		return err
	}

	fmt.Printf("The value of sdk inside the fabric.go%v", useVolume)
	// time.Sleep(2 * time.Second)
	if useSDK {
		// f.createChannel(userId, useSDK)
		f.fetchNetworkConfigFile(userId)

	} else {
		f.createChannel(userId, useVolume)
		f.joinChannel(userId, useVolume)
		f.getBlockInformation(userId, useVolume)

		// Fetching the ccp file.

		//Use this section to define network using the SDK.
		// First step is to access the ccp file for configuration.
		// Check for the error.
		// Since the folders are present in the orgs structure -> which are located inside the container special precaution must be take when using that.

	}

	// Next step is to actually run the container and pass the parameter in the containers.
	// For this particular use case we will get hte docker instance of the machine and then run the container in the fabric_docker file.
	// This container start up can be different according to the container so for example the startup function in the deployerinterface should be created.

	// Currently i am planning to use the functions the docker code from the firefly cli seems quite nice way of handling things.
	return nil
}

func (f *FabricDefinition) Join(networkId string, orgName string, networkId2 string, joiningOrgName string, userid string, useVolume bool, finalizePhase bool) (err error) {
	// Starting step would be to check if the network is already present and if so then it would kind of load the network.(Dont exactly know how it should load the network)
	// The first step can be to try to get the location for the org and then using the tool to generate the files needed.
	// the files which are needed are the crypto, configtx, docker-compose.
	// Currently we will just load these files and not create them.
	userIdentification = userid
	verbose = true
	f.UseVolume = useVolume
	f.Deployer = getDeployerInstance(f.DeployerType)
	// First checking if the network is already present, other wise creating the org3 structure

	// THis is for creating the other organization, which is not needed now.

	// The previous step should be asynchronous though

	// Steps done by the Organization after
	if !finalizePhase {
		f.createOrganizationForJoin(userid, networkId, orgName)
		f.fetchConfigBlock(userid)

		f.envelopeBlockCreation(userid, networkId, orgName)
		f.signConfig(fmt.Sprintf("%s_update_in_envelope.pb", orgName))
		f.signAndUpdateConfig(fmt.Sprintf("%s_update_in_envelope.pb", orgName))

		f.fetchChannelGenesisBlock()
		f.loadGenesisFileToOrg(networkId)
	} else {
		workingDir := path.Join(constants.EnablerDir, userid, networkId)

		f.Deployer.Deploy(workingDir)

		f.joinOtherOrgPeerToChannel(userid, networkId, orgName)

		f.createAnchorPeer(userid, networkId, orgName)
	}

	// Run the docker compose from the org3 -> container
	//  Next join the channel from org3 peer.

	// Bring up the docker compose file , the container and try to join the channel using the container
	// Currently there is a probelm with the peer channel fetch config -|> As the config is unable to be fetched
	// Next steps would be to add the anchor peer for the organization 3.

	// So go to the folder structure and then create a docker instance and then create a volume, copy files in the volume. -> crypto, configtx
	// Once volume is done then create the crypto files using crypto command.
	return nil
}
func (f *FabricDefinition) Leave(networkId string, orgName string, userId string, useVolume bool) error {
	userIdentification = userId
	verbose = true
	f.UseVolume = useVolume
	// THe file is generated by the org3 after it has joined the network.
	f.leaveNetwork(userId, networkId, orgName)
	// The file needs to be signed by the others and uploaded to the
	f.signConfig("config_update_in_envelope.pb")
	f.signAndUpdateConfig("config_update_in_envelope.pb")

	return nil
}

// This functin creates org3 as for joining and along with that generates the definition file for the org3.
func (f *FabricDefinition) createOrganizationForJoin(userId string, networkId string, orgName string) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, networkId, "blockchain")
	configtxPath := path.Join(blockchainDirectory, "configtx.yaml")
	volumeName := fmt.Sprintf("%s_fabric", networkId)
	enablerPath := path.Join(constants.EnablerDir, userId, networkId, "enabler")
	f.Logger.Printf("Generating the volume with volume name: %s", volumeName)
	var storageType string
	if f.UseVolume {
		storageType = volumeName
		if err := docker.CreateVolume(volumeName, verbose); err != nil {
			return err
		}
	} else {
		storageType = enablerPath
	}

	// Running the configtxgen command to get the org3 definition and store it in the current folder.
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", configtxPath, storageType, orgName, enablerPath, orgName))

	fmt.Printf(" %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Printf("%s", out)

	return nil
}

// This is in the Preparation phase for join
//
// This function fetches the latest configuration block from the channel and then transforms it into a json format while extracting the necessary structure.
func (f *FabricDefinition) fetchConfigBlock(userId string) (err error) {
	var storageType string
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(f.Enabler.Members[0].OrgName), f.Enabler.Members[0].DomainName)
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	f.Logger.Printf("Fetching config block for channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", "enablerchannel", "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", f.Enabler.NetworkName, storageType, networkDir))

	out, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf(" %s\n", out)
	return nil
}

func (f *FabricDefinition) createAnchorPeer(userID string, networkId string, orgName string) (err error) {
	f.Logger.Printf("Creating anchor peers block for channel")
	networkDir := path.Join(constants.EnablerDir, userID, networkId)
	var storageType string
	var channelName string
	channelName = "enablerchannel"
	domainName := "example.com"
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(orgName), domainName)
	peerID := fmt.Sprintf("%s.%s.%s", "peer0", strings.ToLower(orgName), domainName)
	enablerPath := path.Join(constants.EnablerDir, userID, networkId, "enabler")

	volumeName := fmt.Sprintf("%s_fabric", networkId)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", fmt.Sprintf("%s", channelName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", f.Enabler.NetworkName, storageType, networkDir)).Output()
	if err != nil {
		return err
	}
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq '.channel_group.groups.Application.groups.%sMSP.values += {\"AnchorPeers\":{\"mod_policy\": \"Admins\",\"value\":{\"anchor_peers\": [{\"host\": \"%s\",\"port\": 7051}]},\"version\": \"0\"}}' /etc/enabler/config1.json  > %s/enabler/modified_anchor_config.json ", f.Enabler.NetworkName, storageType, networkDir, orgName, peerID, networkDir)).Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/modified_anchor_config.json:/etc/enabler/modified_anchor_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_anchor_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_anchor_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", fmt.Sprintf("%s", channelName), "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_anchor_config.pb", "--output", "/etc/enabler/anchor_update.pb")
	// Required Step
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/anchor_update.pb --type common.ConfigUpdate | jq . > %s/enabler/anchor_update.json", f.Enabler.NetworkName, storageType, networkDir)).Output()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/anchor_update.json)'}}}'| jq . > %s/enabler/anchor_update_in_envelope.json", f.Enabler.NetworkName, storageType, channelName, networkDir, networkDir))
	fmt.Printf("%s", cmd.String())
	out, err = cmd.Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/anchor_update_in_envelope.json:/etc/enabler/anchor_update_in_envelope.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/anchor_update_in_envelope.json", "--type", "common.Envelope", "--output", "/etc/enabler/anchor_update_in_envelope.pb")

	// Before doing this need to copy the cafile from the orderer msp-> tlsca .pem to org3 accessible location
	// Then only it would work.
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/tlsca.example.com-cert.pem:/etc/enabler/tlsca.example.com-cert.pem", networkDir), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "update", "-f", "/etc/enabler/anchor_update_in_envelope.pb", "-c", fmt.Sprintf("%s", channelName), "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("%s/tlsca.%s-cert.pem", "/etc/enabler", f.Enabler.Members[0].DomainName))

	fmt.Printf(" %s\n", out)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "getinfo", "-c", "enablerchannel")

}

func (f *FabricDefinition) leaveNetwork(userID string, networkId string, orgName string) (err error) {
	f.Logger.Printf("Leaving the network")
	networkDir := path.Join(constants.EnablerDir, userID, networkId)
	parentNetworkDir := path.Join(constants.EnablerDir, userID, f.Enabler.NetworkName)
	var storageType string
	volumeName := fmt.Sprintf("%s_fabric", networkId)
	enablerPath := path.Join(constants.EnablerDir, userID, networkId, "enabler")
	channelName := "enablerchannel"
	domainName := "example.com"
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(orgName), domainName)
	peerID := fmt.Sprintf("%s.%s.%s", "peer0", strings.ToLower(orgName), domainName)

	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}

	parentVolumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", fmt.Sprintf("%s", channelName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", f.Enabler.NetworkName, storageType, networkDir)).Output()
	if err != nil {
		return err
	}
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq 'del(.channel_group.groups.Application.groups.%sMSP)' /etc/enabler/config1.json  > %s/enabler/modified_config.json ", f.Enabler.NetworkName, storageType, networkDir, orgName, networkDir)).Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", "enablerchannel", "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_config.pb", "--output", "/etc/enabler/config_update.pb")
	// Required Step
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/config_update.pb --type common.ConfigUpdate | jq . > %s/enabler/config_update.json", f.Enabler.NetworkName, storageType, networkDir)).Output()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/config_update.json)'}}}'| jq . > %s/enabler/config_update_in_envelope.json", f.Enabler.NetworkName, storageType, channelName, networkDir, networkDir))
	fmt.Printf("%s", cmd.String())
	out, err = cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf(" %s\n", out)

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config_update_in_envelope.json:/etc/enabler/config_update_in_envelope.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config_update_in_envelope.json", "--type", "common.Envelope", "--output", "/etc/enabler/config_update_in_envelope.pb")

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "signconfigtx", "-f", fmt.Sprintf("/etc/enabler/%s", "config_update_in_envelope.pb"), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	if f.UseVolume {
		docker.CopyFromContainer(fmt.Sprintf("%s", peerID), "/etc/enabler/config_update_in_envelope.pb", fmt.Sprintf("%s/enabler/config_update_in_envelope.pb", networkDir), verbose)

		docker.CopyFileToVolume(parentVolumeName, fmt.Sprintf("%s/enabler/config_update_in_envelope.pb", networkDir), fmt.Sprintf("config_update_in_envelope.pb"), verbose)
	} else {
		docker.CopyFromContainer(fmt.Sprintf("%s", peerID), "/etc/enabler/config_update_in_envelope.pb", fmt.Sprintf("%s/enabler/config_update_in_envelope.pb", parentNetworkDir), verbose)

	}

	return nil
	// Now sign this transaction and update it.
	// Before doing this need to copy the cafile from the orderer msp-> tlsca .pem to org3 accessible location
	// Then only it would work.
	// docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s/enabler/tlsca.example.com-cert.pem:/etc/enabler/tlsca.example.com-cert.pem", networkDir), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
	// 	"CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org3.example.com/peers/fabric_peer.org3.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org3MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp", "hyperledger/fabric-tools:2.3",
	// 	"peer", "channel", "update", "-f", "/etc/enabler/anchor_update_in_envelope.pb", "-c", "enablerchannel", "-o", "fabric_orderer:7050", "--tls", "--cafile", fmt.Sprintf("%s/tlsca.example.com-cert.pem", "/etc/enabler"))

	// return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e", "CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org3.example.com/peers/fabric_peer.org3.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org3MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp", "hyperledger/fabric-tools:2.3", "peer", "channel", "getinfo", "-c", "enablerchannel")

}

// This is in the preparation phase
// This function uses the definition file provided by the organization which wishes to join the network and then uses this file to create an envelope, containing the info to join the network.
func (f *FabricDefinition) envelopeBlockCreation(userId string, networkId string, orgName string) (err error) {
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	// Required Step
	var storageType string
	var channelName string
	channelName = "enablerchannel"
	domainName := "example.com"

	peerID := fmt.Sprintf("%s.%s.%s", "peer0", strings.ToLower(f.Enabler.Members[0].OrgName), domainName)
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	orgDefFilePath := path.Join(path.Join(constants.EnablerDir, userId, networkId, "enabler"), fmt.Sprintf("%s.json", orgName))
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}
	// Required Step
	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s:/etc/enabler/%s.json -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq -s '.[0] * {\"channel_group\":{\"groups\":{\"Application\":{\"groups\": {\"%sMSP\":.[1]}}}}}' /etc/enabler/config1.json /etc/enabler/%s.json > %s/enabler/modified_config.json ", f.Enabler.NetworkName, storageType, orgDefFilePath, orgName, networkDir, orgName, orgName, networkDir)).Output()
	if err != nil {
		return err
	}
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", "enablerchannel", "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_config.pb", "--output", fmt.Sprintf("/etc/enabler/%s_update.pb", orgName))
	// Required Step
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/%s_update.pb --type common.ConfigUpdate | jq . > %s/enabler/%s_update.json", f.Enabler.NetworkName, storageType, orgName, networkDir, orgName)).Output()

	// Required Step

	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/%s_update.json)'}}}'| jq . > %s/enabler/%s_update_in_envelope.json", f.Enabler.NetworkName, storageType, channelName, networkDir, orgName, networkDir, orgName))

	fmt.Printf("%s", cmd.String())
	out, err = cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf(" Printng out %s\n", out)

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/%s_update_in_envelope.json:/etc/enabler/%s_update_in_envelope.json", networkDir, orgName, orgName), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", fmt.Sprintf("/etc/enabler/%s_update_in_envelope.json", orgName), "--type", "common.Envelope", "--output", fmt.Sprintf("/etc/enabler/%s_update_in_envelope.pb", orgName))

	// copying  the output .pb file into the directory.
	docker.CopyFromContainer(fmt.Sprintf("%s", peerID), fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName), fmt.Sprintf("%s/tlsca.%s-cert.pem", path.Join(constants.EnablerDir, userId, networkId, "enabler"), f.Enabler.Members[0].DomainName), verbose)
	docker.CopyFromContainer(fmt.Sprintf("%s", peerID), fmt.Sprintf("/etc/enabler/%s_update_in_envelope.pb", orgName), fmt.Sprintf("%s/enabler/%s_update_in_envelope.pb", networkDir, orgName), verbose)
	// copying  the output .pb file into the directory.
	// docker.CopyFromContainer(fmt.Sprintf("%s_fabric_peer", f.Enabler.NetworkName), "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem", fmt.Sprintf("%s/tlsca.example.com-cert.pem", enablerPath), verbose)
	// docker.CopyFromContainer(fmt.Sprintf("%s_fabric_peer", f.Enabler.NetworkName), fmt.Sprintf("/etc/enabler/%s_update_in_envelope.pb", orgName), fmt.Sprintf("%s/enabler/%s_update_in_envelope.pb", networkDir, orgName), verbose)
	return nil
}

func (f *FabricDefinition) signConfig(envelopeFile string) error {
	f.Logger.Printf("Signing the config block for channel")
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	if f.UseVolume {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "signconfigtx", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	} else {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "signconfigtx", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	}

	return nil
}
func (f *FabricDefinition) signByOrg3Config(envelopeFile string, networkName string) error {
	f.Logger.Printf("Signing the config block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, networkName)
	volumeName := fmt.Sprintf("%s_fabric", networkName)
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		"CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3",
		"peer", "channel", "signconfigtx", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "--tls", "--cafile", "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem")

	return nil
}

func (f *FabricDefinition) signAndUpdateConfig(envelopeFile string) error {
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	f.Logger.Printf("Sign and Update config block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", "enablerchannel", "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	} else {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", "enablerchannel", "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	}

	return nil
}

// This function fetches the gensis block, and then copies the genesis block to the org which requested for it.
// NOTE: The request still needs to be done currently only copy is being done and the genesis block is copied onto the requesting organization's folder/volume.
func (f *FabricDefinition) fetchChannelGenesisBlock() error {
	var storageType string
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	f.Logger.Printf("Fetching channel genesis block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	if f.UseVolume {
		storageType = volumeName

	} else {
		storageType = enablerPath
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "0", "/etc/enabler/channel_genesis_block.block", "-c", "enablerchannel", "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	// copy the file channel_genesis_block.block to the another org

	docker.CopyFromContainer(peerID, "/etc/enabler/channel_genesis_block.block", fmt.Sprintf("%s/enabler/channel_genesis_block.block", networkDir), verbose)
	return nil
}
func (f *FabricDefinition) loadGenesisFileToOrg(networkId string) error {
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	enablerPath := path.Join(constants.EnablerDir, userIdentification, networkId, "enabler")
	destinationFile := path.Join(enablerPath, "channel_genesis_block.block")
	if f.UseVolume {
		volumeName := fmt.Sprintf("%s_fabric", networkId)
		// enablerPath := path.Join(constants.EnablerDir, userIdentification, networkId, "enabler")

		docker.CopyFileToVolume(volumeName, fmt.Sprintf("%s/enabler/channel_genesis_block.block", networkDir), fmt.Sprintf("channel_genesis_block.block"), verbose)

	} else {
		input, err := ioutil.ReadFile(fmt.Sprintf("%s/enabler/channel_genesis_block.block", networkDir))
		if err != nil {
			fmt.Println(err)
			return err
		}

		err = ioutil.WriteFile(destinationFile, input, 0644)
		if err != nil {
			fmt.Println("Error creating", destinationFile)
			fmt.Println(err)
			return err
		}
	}

	// docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", "CORE_PEER_ADDRESS=fabric_peer:7051", "-e", "CORE_PEER_TLS_ENABLED=true", "-e", "CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/enabler.block")
	return nil
}

// This function fetches the ccp config file from the folder of the network and then calls the fabric-sdk-go config for generating the configoption/ configprovider using this file.
func (f *FabricDefinition) fetchNetworkConfigFile(userId string) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	enablerDirectoryPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	ccpFilePath := path.Join(blockchainDirectory, "ccp.yaml")
	if err := createNewContext(config.FromFile(ccpFilePath), enablerDirectoryPath); err != nil {
		fmt.Printf("Printing error%s", err)
	}
	// Now since we have the ccp file path, need to call the fabric sdk go lang.
	return nil
}

func createNewContext(configProvider core.ConfigProvider, blockchainDirectoryPath string) (err error) {
	var fabricSDK *fabsdk.FabricSDK
	var resclient *resmgmt.Client
	sdk, err := fabsdk.New(configProvider)
	if err != nil {
		return errors.WithMessage(err, "failed to create SDK")
	}
	fabricSDK = sdk
	// Fabric sdk create
	// Next the resource management client needed for managing channels.

	// Msp client allows to retrieve user info from identity as signing identity -> need to save the channel.

	resourceManagerClientContext := fabricSDK.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg("org1.example.com"))
	resMgmtClient, err := resmgmt.New(resourceManagerClientContext)
	if err != nil {
		fmt.Printf("failed to create channel management client from Admin identity %s", err)
	}
	resclient = resMgmtClient
	fmt.Println("Resource Management client created.")
	// Generate msp client for the context
	mspClient, err := mspclient.New(resourceManagerClientContext, mspclient.WithOrg("org1.example.com"))
	if err != nil {
		return errors.WithMessage(err, "failed to create MSP client")
	}
	// use the mspclient to get signing identity for admin user.
	adminId, err := mspClient.GetSigningIdentity("Admin")
	if err != nil {
		return errors.WithMessage(err, "failed to get admin signing identity")
	}

	// structure the save channel request. and then execute the request.

	saveChannelRequest := resmgmt.SaveChannelRequest{
		ChannelID:         "enablerchannel",
		ChannelConfigPath: path.Join(blockchainDirectoryPath, "enabler.tx"),
		SigningIdentities: []msp.SigningIdentity{adminId},
	}

	txnId, err := resclient.SaveChannel(saveChannelRequest, resmgmt.WithOrdererEndpoint("fabric_orderer"))

	if err != nil || txnId.TransactionID == "" {
		return errors.WithMessage(err, "failed to save channel")
	}
	fmt.Println("Channel created")

	// Now structure and execute the request.

	// fmt.Println("%s", resclient)
	// clientContext := sdk.Context()

	return nil

}

// setting up the docker container and the volume and running the cryptogen configs
func (f *FabricDefinition) generateCryptoMaterial(userId string, useVolume bool) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)

	// volumeName := fmt.Sprintf("enabler_fabric")
	f.Logger.Printf("Generating the volume with volume name: %s", volumeName)
	if err := docker.CreateVolume(volumeName, verbose); err != nil {
		return err
	}
	f.Logger.Printf("Using the fabric tools to generate the msp with cryptogen tool in the shared volume location")
	// Run cryptogen to generate MSP
	if !useVolume {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
			return err
		}
	} else {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
			return err
		}
	}
	fmt.Printf("Check for network")
	if err := docker.InspectNetwork(fmt.Sprintf("%s_default", f.Enabler.NetworkName), true); err != nil {
		docker.CreateNetwork(fmt.Sprintf("%s_default", f.Enabler.NetworkName), true)
	}

	return nil
}

func (f *FabricDefinition) generateGenesisBlock(userId string, useVolume bool) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	f.Logger.Printf("Using the fabric tools to generate the Gensis block in the shared volume location")
	// Generate genesis block
	// might also need to generate the configtx yaml file according the orgname and even the name as example.com does not seem quite good enough
	fmt.Printf("Location of the configtx file %s", path.Join(blockchainDirectory, "configtx.yaml"))
	if !useVolume {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", "enablerchannel"); err != nil {
			//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
			return err
		}
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputCreateChannelTx", "/etc/enabler/enabler.tx", "-profile", "SingleOrgApplicationGenesis", "-channelID", "enablerchannel"); err != nil {
			//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
			return err
		}
	} else {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", "enablerchannel"); err != nil {
			//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
			return err
		}
		// if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputCreateChannelTx", "/etc/enabler/enabler.tx", "-profile", "Channel1", "-channelID", "enablerchannel"); err != nil {
		// 	//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
		// 	return err
		// }
		// if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputAnchorPeersUpdate", "/etc/enabler/Org1MSPanchors.tx", "-profile", "Channel1", "-channelID", "enablerchannel", "-asOrg", "Org1MSP"); err != nil {
		// 	//  "-outputCreateChannelTx", "create_chan_tx.pb", "-printOrg", "Org1",
		// 	return err
		// }
		// docker.CopyFromContainer(fmt.Sprintf("%s_fabric_orderer", f.Enabler.NetworkName), "/etc/enabler/genesis.block", fmt.Sprintf("%s/genesis.block", enablerDirectory), verbose)
	}

	return nil
}

func (f *FabricDefinition) createChannel(userId string, useVolume bool) (err error) {
	verbose := true
	f.Logger.Printf("Creating channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if !useVolume {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "hyperledger/fabric-tools:2.3", "osnadmin", "channel", "join", "--channelID", "enablerchannel", "--config-block", "/etc/enabler/enabler.block", "-o", fmt.Sprintf("%s:7053", f.Enabler.Members[0].OrdererName), "--ca-file", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/ca.crt", f.Enabler.Members[0].DomainName), "--client-cert", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.crt", f.Enabler.Members[0].DomainName), "--client-key", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.key", f.Enabler.Members[0].DomainName))

	} else {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "osnadmin", "channel", "join", "--channelID", "enablerchannel", "--config-block", "/etc/enabler/enabler.block", "-o", fmt.Sprintf("%s:7053", f.Enabler.Members[0].OrdererName), "--ca-file", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/ca.crt", f.Enabler.Members[0].DomainName), "--client-cert", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.crt", f.Enabler.Members[0].DomainName), "--client-key", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.key", f.Enabler.Members[0].DomainName))

	}
	// volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	// docker.CopyFromContainer(fmt.Sprintf("fabric_orderer"), "/etc/enabler/genesis.block", fmt.Sprintf("%s/enabler/genesis.block", networkDir), verbose)

	// return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain"), "configtx.yaml")), "-e", "CORE_PEER_ADDRESS=peer0.org1.example.com:7051","-e","SYS_CHANNEL=channel1", "-e", "CORE_PEER_TLS_ENABLED=true", "-e", "CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3", "peer", "channel", "create", "-c", "channel1", "-f", "/etc/enabler/enabler.tx", "-o", fmt.Sprintf("fabric_orderer:7050"), "--tls", "--cafile", "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem")

}

func (f *FabricDefinition) joinChannel(userId string, useVolume bool) error {
	verbose := true
	f.Logger.Printf("Joining channel")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)

	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(f.Enabler.Members[0].OrgName), f.Enabler.Members[0].DomainName)
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	if useVolume {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/enabler.block")
	} else {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/enabler.block")
	}

}

func (f *FabricDefinition) joinOtherOrgPeerToChannel(userId string, networkId string, orgName string) error {
	verbose := true
	f.Logger.Printf("Joining channel")
	channelName := "enablerchannel"
	volumeName := fmt.Sprintf("%s_fabric", networkId)
	enablerDirectory := path.Join(constants.EnablerDir, userId, networkId, "enabler")
	var storageType string
	networkDir := path.Join(constants.EnablerDir, userId, networkId)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerDirectory
	}
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(orgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/channel_genesis_block.block")
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", orgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "getinfo", "-c", fmt.Sprintf("%s", channelName))
}

func (f *FabricDefinition) getBlockInformation(userId string, useVolume bool) error {
	verbose := true
	f.Logger.Printf("Get block information")
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	orgName := f.Enabler.Members[0].OrgName
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	blockchainDir := path.Join(networkDir, "blockchain")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	var cmd *exec.Cmd
	if useVolume {
		cmd = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s/configtx.yaml:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", blockchainDir, volumeName, orgName, enablerDirectory, orgName))
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "configtxlator", "proto_decode", "--input", "/etc/enabler/enabler.block", "--output", "/etc/enabler/enabler.json", "--type", "common.Block")

	} else {
		cmd = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s/configtx.yaml:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", blockchainDir, enablerDirectory, orgName, enablerDirectory, orgName))

		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "hyperledger/fabric-tools:2.3", "configtxlator", "proto_decode", "--input", "/etc/enabler/enabler.block", "--output", "/etc/enabler/enabler.json", "--type", "common.Block")

	}
	fmt.Printf(" %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Printf("%s", out)

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
