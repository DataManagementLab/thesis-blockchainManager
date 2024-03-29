package fabric

import (
	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	"gopkg.in/yaml.v2"

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

type QueryInstalledChaincodes struct {
	InstalledChaincodes []*InstalledChaincode `json:"installed_chaincodes"`
}

type InstalledChaincode struct {
	PackageID string `json:"package_id,omitempty"`
	Label     string `json:"label,omitempty"`
}

var fab *FabricDefinition

//go:embed configtx.yaml
var configtxYaml string

//go:embed configtx-basicsetup.yaml
var configtxBasicSetupYaml string

//// go:embed chaincode/chaincode.go
var chaincodeImplementation string

var userIdentification string
var verbose bool

// Init is the implemenation of the init function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Generating the necessary file for deployment, configuration and cryptographic material.
func (f *FabricDefinition) Init(userId string, useVolume bool, basicSetup bool, localSetup bool, logging bool) (err error) {

	//Steps to follow:
	// Basic step to fetch the deployer instance.
	// call the deployer init function then -> deployer init will create the dockercompose basic setup.
	// 1.Creating docker compose
	// 2. ensure directories
	// 3. write configs
	// 4. write docker compose

	// Current decision is to take the docker as default deployment platform.

	// check if the fabric deployertype is docker then initialze deployer with it.
	verbose = logging
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
	if err := f.generateCryptoMaterial(userId, useVolume, localSetup); err != nil {
		return err
	}
	if !basicSetup {
		fmt.Printf("\n\nThe user '%s' has been Successfully initialized. To create the network, run:\n\n go run main.go create -u %s\n\n", userId, userId)
	} else {
		fmt.Printf("\n\nThe user '%s' has been Successfully initialized. This can directly be used in join to join an already existing network. Checkout join command for more information.\n ", userId)
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
	platformDir := filepath.Join(constants.EnablerDir)
	dir, err := os.ReadDir(platformDir)
	if err != nil {
		panic(err)
	}
	count := len(dir)
	// fmt.Println(" The current count", count)
	external := map[string]int{
		"ca_server_port":                       7054,
		"ca_operations_listen_port":            17054,
		"orderer_general_listen_port":          7050,
		"orderer_admin_listen_port":            7053,
		"orderer_operations_listen_port":       17050,
		"core_peer_listen_address_gossip_port": 7051,
		"core_peer_chaincode_listen_port":      7052,
		"core_operations_listen_port":          17051,
		"core_peer_listen":                     7051 + count*100,
		"core_peer_operation":                  17051 + count*100,
		"orderer_general":                      7050 + count*100,
		"orderer_admin":                        7053 + count*100,
		"orderer_operations":                   17050 + count*100,
		"ca_server":                            7054 + count*100,
		"ca_operations":                        17054 + count*100,
	}
	return external

}

func getDeployerInstance(deployerType string) (deployer deployer.IDeployer) {
	if deployerType == "docker" {
		return GetFabricDockerInstance()
	}
	return GetFabricDockerInstance()
}

// Create is the implemenation of the create function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Creating the genesis block
// 2. Deploying the containers for the network
// 3. Packaging and installing the chaincode implementation on the peers
// 4. Creating the channel.
// 5. Making the peers for the organization join the channel.
// 6. Fetching the block information and fetching the genesis block details.
func (f *FabricDefinition) Create(userId string, useSDK bool, useVolume bool, logging bool) (err error) {
	// Step to do inside the create function

	// 1.Also need to check if the docker is present in the host machine.
	// 2. We would need to run the first time setup where the initiailization of blockcahin node happens.
	verbose = logging
	f.Deployer = getDeployerInstance(f.DeployerType)
	userIdentification = userId
	workingDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)

	f.generateGenesisBlock(userId, useVolume)

	if err := f.Deployer.Deploy(workingDir, verbose); err != nil {
		return err
	}

	packageChaincodeImplementation(filepath.Join(workingDir, "enabler"))
	f.createChannel(userId, useVolume)
	f.Logger.Printf("Channel Creation done successfully .")
	f.joinChannel(userId, useVolume)

	// After the join channel part is done can implement the chaincode deployment, however , we can use a method to do the deployment in the blockchain enabler interface.

	// Now in order to deploy the chaincode.
	// Steps are needed:
	// 1. Package the chaincode -> take the chaincode folder and then package the contents of this folder.
	// 2. After this install the packaged chaincode
	// 3. query the installed chaincode
	// 4. approve the chaincode.
	// 5. commit the chaincoe.

	f.getBlockInformation(userId, useVolume)
	f.fetchChannelGenesisBlock()

	return nil
}

// This function::packageChaincodeImplementation is required to copy the chaincode into the correct folder
func packageChaincodeImplementation(enablerPath string) {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	chaincodeDir := filepath.Join(currentPath, "/BlockchainEnabler/internal/blockchain/fabric/chaincode")

	cmd := exec.Command("cp", "-R", chaincodeDir, enablerPath)
	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}
}

// Add is the implemenation of the add function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Fetching the config block for the network.
// 2. Loading passed invite file and unzipping and storing it within own folder structure.
// 3. Transforming the definition file.
// 4. Creating the envelope file and signing it.
// 5. Updating this signed envelope file to the networks or providing user message to send it to other organization part of the network.
func (f *FabricDefinition) Add(userid string, useVolume bool, zipfile string, logging bool) (err error) {
	userIdentification = userid
	verbose = logging
	// var networkDetails  *types.FabricDefinition
	f.UseVolume = useVolume
	var blockchaindefinition interface{}
	var ownBlockchainDefinition interface{}
	f.Deployer = getDeployerInstance(f.DeployerType)
	f.fetchConfigBlock(userid)
	zipFile := filepath.Base(zipfile)
	zipFileSplit := strings.Split(strings.TrimSuffix(zipFile, filepath.Ext(zipFile)), "_")

	enablerPath := filepath.Join(constants.EnablerDir, userid, f.Enabler.NetworkName, "enabler")
	pathUser := filepath.Join(enablerPath, zipFileSplit[0])

	// convert or transform this file specified in the path above

	// here it needs to copy the zip file unpack it load it into another folder and use the information provided in that folder -> read the network_config.json file.
	f.unzipFile(zipfile, userid, zipFileSplit[0])
	networkConfig := f.loadNetworkConfig(fmt.Sprintf("%s", filepath.Join(pathUser, "network_config.json")), userid)
	ownNetworkConfig := f.loadNetworkConfig(fmt.Sprintf("%s", filepath.Join(enablerPath, "network_config.json")), userid)
	// next load this file
	blockchaindefinition = &networkConfig.BlockchainDefinition
	networkDetails, ok := blockchaindefinition.(*types.FabricDefinition)
	if ok {
		networkId := networkConfig.NetworkName
		orgName := networkDetails.OrganizationInfo.OrganizationName

		f.transformDefinitionFile(filepath.Join(pathUser, fmt.Sprintf("%s.json", orgName)), orgName, userid)
		// Now use this file from that location,
		// add the signature, from multiple parties -> for thia just need to use the sign command & update command.
		f.Logger.Printf("Adding the organization to the Network . . . this might take few seconds. ")
		if err := f.envelopeBlockCreation(userid, networkId, orgName); err != nil {
			return err
		}
		if err := f.signConfig(fmt.Sprintf("%s_update_in_envelope.pb", orgName)); err != nil {
			return err
		}
		ownBlockchainDefinition = &ownNetworkConfig.BlockchainDefinition
		ownNetworkDetails, ok := ownBlockchainDefinition.(*types.FabricDefinition)
		if ok {
			if len(ownNetworkDetails.NetworkMembers) > 1 {
				// ALso need to create the zip file here with the network config, envelope file -> json and pb format.
				cafile := fmt.Sprintf("organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName)

				createZipForSign(enablerPath, fmt.Sprintf("%s_update_in_envelope.pb", orgName), fmt.Sprintf("signed_%s_update_in_envelope.json", orgName), filepath.Join(enablerPath, "network_config.json"), cafile, orgName, f.Enabler.NetworkName)
				f.Logger.Printf("Organization needs the signature of others to be added to the Network. ")
				printNetworkMembers(orgName, ownNetworkDetails.NetworkMembers)
				// fmt.Printf("Need to use the sign command to send the zip file %s_sign_transfer.zip to %v", orgName, &ownNetworkDetails.NetworkMembers)
			} else {
				channelName := f.Enabler.Members[0].ChannelName
				if err := f.signAndUpdateConfig(fmt.Sprintf("%s_update_in_envelope.pb", orgName), channelName); err != nil {
					return err
				}
				ownNetworkDetails.NetworkMembers = append(ownNetworkDetails.NetworkMembers, &orgName)

				netConfig := types.NetworkConfig{
					NetworkName:          ownNetworkConfig.NetworkName,
					BlockchainDefinition: ownNetworkConfig.BlockchainDefinition,
				}
				content, err := json.MarshalIndent(netConfig, "", " ")
				if err != nil {
					fmt.Println(err)
				}
				err = writeNetworkConfig(userid, ownNetworkConfig.NetworkName, content)
				if err != nil {
					fmt.Println(err)
					return err
				}

				f.Logger.Printf("Network has been updated for the organization to join.")
				fmt.Printf("\n\nThe organization %s has been updated to the network, Use the join command to join the network.\n\n", orgName)
				// write network config.
			}
		}
		// Need to use the configuration from the network config -> and then add the organzation to the list of participants.
		//  the sign and update only happens if either all the participants signatures are there already.
		// In sign would need to check the signatures that are present against the list of participants.
		// f.signAndUpdateConfig(fmt.Sprintf("%s_update_in_envelope.pb", orgName))
	} else {
		fmt.Printf("An error occured %s", blockchaindefinition)
	}
	return nil
}

func printNetworkMembers(orgName string, networkMembers []*string) {
	fmt.Printf("\n\nNeed to use the sign command to send the zip file %s_sign_transfer.zip to ", orgName)
	for _, u := range networkMembers {
		fmt.Printf(" %v ", *u)

	}
	fmt.Println()
}
func writeNetworkConfig(userId string, networkName string, content []byte) error {
	if err := ioutil.WriteFile(filepath.Join(constants.EnablerDir, userId, networkName, "enabler", fmt.Sprintf("network_config.json")), content, 0755); err != nil {
		return err
	}
	return nil
}

// Sign is the implemenation of the sign function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Loads the zip file passed in the argument.
// 2. Signs the envelope file containing other signatures.
// 3. Update it to the network or ask user to send it to other organizations part of the network.
func (f *FabricDefinition) Sign(userid string, useVolume bool, zipfile string, update bool, logging bool) (err error) {

	userIdentification = userid
	verbose = logging
	var channelName string
	var networkDetails *types.FabricDefinition
	var blockchaindefinition interface{}
	f.UseVolume = useVolume
	var networkName string
	var ordererName string
	var cafile string
	// takes in the zip file, checks participant list and signature, then signs it and uploads it / gives message with generated zip file to send to another org,
	// read the zip file and take the .pb and .json files also identify the networkconfig file.

	// finding the filename for the zip file.

	zipFile := filepath.Base(zipfile)
	zipFileSplit := strings.Split(strings.TrimSuffix(zipFile, filepath.Ext(zipFile)), "_")

	f.unzipFile(zipfile, userid, zipFileSplit[0])
	enablerPath := filepath.Join(constants.EnablerDir, userid, f.Enabler.NetworkName, "enabler")
	signPath := filepath.Join(enablerPath, zipFileSplit[0])

	// go to the path for sign and then fetch the .pb file, transform, copy the file into the enabler path.

	envelopeName := findFileName(signPath, ".pb")
	fmt.Printf("Envelope name %v ", envelopeName)
	dstPath := path.Join(enablerPath, fmt.Sprintf(envelopeName[0]))
	transformFile(filepath.Join(signPath, envelopeName[0]), dstPath)
	envelopeNameWithoutExt := strings.TrimSuffix(envelopeName[0], filepath.Ext(envelopeName[0]))
	networkConfig := f.loadNetworkConfig(fmt.Sprintf("%s", filepath.Join(signPath, "network_config.json")), userid)
	// // next load this file
	blockchaindefinition = &networkConfig.BlockchainDefinition
	networkDetails, ok := blockchaindefinition.(*types.FabricDefinition)
	// signedEnvelopePath := filepath.Join(pathUser, "signed_envelope.json")
	if ok {
		channelName = networkDetails.OrganizationInfo.ChannelName
		networkName = networkConfig.NetworkName
		ordererName = networkDetails.OrganizationInfo.OrdererName
		cafile = fmt.Sprintf("%s_tlsca.example.com-cert.pem", networkName)
		transformFile(filepath.Join(signPath, cafile), filepath.Join(enablerPath, cafile))
	}
	f.Logger.Printf("Signing the Transaction ...")
	if !update {
		f.signConfig(fmt.Sprintf(envelopeName[0]))
		createZipForSign(enablerPath, fmt.Sprintf(envelopeName[0]), fmt.Sprintf("signed_%s.json", envelopeNameWithoutExt), filepath.Join(signPath, "network_config.json"), cafile, zipFileSplit[0], networkName)
		f.Logger.Printf("Signature done successfully by the user")
		fmt.Println("\nNeed to send the file to the remaining participants of the network.")
	} else {

		f.Logger.Printf("Updating the transaction on to the network . . .")
		if err := f.signAndUpdateConfigMultipleParticipants(fmt.Sprintf(envelopeName[0]), channelName, networkName, ordererName, cafile); err != nil {
			return err
		}
		f.Logger.Printf("Signature done successfully by the user and the network is ready.")
		fmt.Println("\nSign has been successfully executed and the network is updated, Run the join command to join the network")

	}

	return nil
}

func findFileName(root, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, filepath.Base(s))
		}
		return nil
	})
	return a
}

func loadSignedEnvelope(envelopePath string) {
	var envelope map[string]map[string]map[string]interface{}
	var signatures []Signature
	read, err := ioutil.ReadFile(envelopePath)
	if err != nil {
		log.Fatalf("failed to read file: %s", err)
	}
	json.Unmarshal(read, &envelope)
	signatures = envelope["payload"]["data"]["signatures"].([]Signature)
	// signatures = envelope["payload"]["data"]["signatures"]
	fmt.Printf("%v", signatures)
}

// Delete is the implemenation of the delete function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Stops and removes all the containers used by the network and the organization.
// 2. Clears the folder structure created during the process.
func (f *FabricDefinition) Delete(userId string, logging bool) (err error) {

	//Steps to follow:
	// Basic step to fetch the deployer instance.\return nil
	verbose = logging
	f.Deployer = getDeployerInstance(f.DeployerType)
	userIdentification = userId
	userDir := path.Join(constants.EnablerDir, userId)
	workingDir := path.Join(userDir, f.Enabler.NetworkName)
	// f.Logger.Printf("Removing the Resources  . . .")
	if err := f.Deployer.Terminate(workingDir, verbose); err != nil {
		return err
	}
	f.Logger.Printf("Containers are successfully removed.")
	f.Logger.Printf("Removing the Folder and Infrastructure.")
	err = os.RemoveAll(workingDir)
	if err != nil {
		f.Logger.Printf("User does not have enough previleges to delete this folder, kindly do it manually.")
		return err
	}
	err = os.RemoveAll(userDir)
	if err != nil {
		f.Logger.Printf("User does not have enough previleges to delete this folder, kindly do it manually.")
		return err
	}
	f.Logger.Printf("The resources are cleared successfully.")
	return nil
}

func (f *FabricDefinition) loadNetworkConfig(configFile string, userId string) types.NetworkConfig {

	// var infoFile string

	// infoFile = filepath.Join(constants.EnablerDir, userId, fmt.Sprintf("network_info.json"))
	// can read from the json file outside the names of the networks that are created and then looping through them and opening them.
	// or can use a file which is outside which contains all the info to the different networks and is appended one thing this would do is making things easier while searching for port used.

	var networkConfig *types.NetworkConfig
	read, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("failed to read file: %s", err)
	}
	json.Unmarshal(read, &networkConfig)
	// check for which provider it belongs to.
	// em.logger.Printf("%s",network.NetworkName)
	return *networkConfig
}

func (f *FabricDefinition) transformDefinitionFile(file string, orgName string, userId string) {
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	dstPath := path.Join(enablerPath, fmt.Sprintf("%s.json", orgName))
	transformFile(file, dstPath)

}

func transformFile(sourcePath string, dstPath string) {
	data, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		log.Fatalf("failed reading file: %s", err)
	}
	fileData, err := os.Create(dstPath)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer fileData.Close()

	_, err = fileData.Write(data)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
}

// Join is the implemenation of the join function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. Loads the accept transfer file passed in the argument.
// 2. Fetches the network configurations.
// 3. Updates its own network to accomodate the new network which it wants to join.
// 4. joins the network along with all its peers.
func (f *FabricDefinition) Join(userid string, useVolume bool, zipFile string, basicSetup bool, logging bool) (err error) {
	f.UseVolume = useVolume
	verbose = logging
	var blockchaindefinition interface{}

	workingDir := path.Join(constants.EnablerDir, userid, f.Enabler.NetworkName)
	pathUser := filepath.Join(constants.EnablerDir, userid, f.Enabler.NetworkName, "enabler", userid)

	f.Deployer = getDeployerInstance(f.DeployerType)

	f.unzipFile(zipFile, userid, userid)

	networkConfig := f.loadNetworkConfig(fmt.Sprintf("%s", filepath.Join(pathUser, "network_config.json")), userid)
	// next load this file
	blockchaindefinition = &networkConfig.BlockchainDefinition
	networkDetails, ok := blockchaindefinition.(*types.FabricDefinition)

	// now load the docker compose file, and then add the networks to it.

	// Also need to update the compose file with the network information.
	// if err := f.Deployer.Deploy(workingDir); err != nil {
	// 	return err
	// }

	if ok {
		networkId := networkConfig.NetworkName
		orgName := networkDetails.OrganizationInfo.OrganizationName
		f.Logger.Printf("Loading the Docker Configurations . . .")
		if err := loadComposeFile(path.Join(workingDir, "docker-compose.yml"), networkId, pathUser); err != nil {
			return err
		}
		f.Logger.Printf("Docker Configurations loading Successfully completed !")
		f.Logger.Printf("Docker updating the Network . . .")
		if err := f.Deployer.Deploy(workingDir, verbose); err != nil {
			return err
		}
		f.Logger.Printf("Docker Network updated !")
		time.Sleep(2 * time.Second)
		f.Logger.Printf("Joining the network . . . This might take a few seconds.")
		if err := f.joinOtherOrgPeerToChannel(userid, networkId, orgName); err != nil {
			return err
		}
		f.Logger.Printf("Node is joining the Network . . .")
		if err := f.createAnchorPeer(userid, networkId, orgName); err != nil {
			return err
		}

		f.Logger.Printf("Node successfully Joined the network")
		f.Logger.Printf("Joined the Network !")

		fmt.Printf("\n\n The Organization was able to join the network %s successfully \n \n", networkId)
	} else {
		fmt.Printf("An error occured %s", blockchaindefinition)

	}
	return nil
}

func loadComposeFile(composeFile string, externalNetwork string, pathUser string) error {
	var compose docker.DockerComposeConfig
	var newCompose docker.DockerComposeConfig
	var dockerExtNet docker.DockerNetworkName
	read, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return err
	}
	yaml.Unmarshal(read, &compose)
	var serviceNet []string
	for _, service := range compose.Services {
		serviceNet = service.DockerNetworkNames
		// fmt.Printf("%s -> %s", service.ContainerName, service.DockerNetworkNames)
	}
	serviceNetworks := append(serviceNet, externalNetwork)
	newCompose = compose
	for _, service := range newCompose.Services {
		service.DockerNetworkNames = serviceNetworks
	}

	for _, extnet := range compose.Networks {
		dockerExtNet.DockerExternalNetworkName = extnet.DockerExternalNetwork.DockerExternalNetworkName
		// fmt.Printf("Docker network %s", dockerExtNet.DockerExternalNetworkName)
	}

	dockerNet := docker.DockerNetwork{
		DockerExternalNetwork: &docker.DockerNetworkName{DockerExternalNetworkName: fmt.Sprintf("%s_default", externalNetwork)},
	}
	for _, networks := range serviceNetworks {
		if networks == externalNetwork {
			newCompose.Networks[networks] = &dockerNet
		}
	}
	bytes, err := yaml.Marshal(newCompose)
	if err != nil {
		return err
	}
	ioutil.WriteFile(composeFile, bytes, 0755)
	return nil
}

func (f *FabricDefinition) unzipFile(zipFile string, userId string, destinationFolderName string) {
	dst := destinationFolderName
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")

	ziparchve, err := zip.OpenReader(zipFile)
	if err != nil {
		panic(err)
	}
	dst, err = filepath.Abs(path.Join(enablerPath, dst))
	if err != nil {
		panic(err)
	}
	fmt.Println("dst :", dst)

	// fmt.Println(" file path",zipFile)
	defer ziparchve.Close()
	for _, f := range ziparchve.File {
		fileLocation := path.Join(dst, f.Name)
		if !strings.HasPrefix(fileLocation, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory...")
			os.MkdirAll(fileLocation, 0777)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fileLocation), 0777); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(fileLocation, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())

		if err != nil {
			panic(err)
		}
		defer dstFile.Close()
		zippedFile, err := f.Open()
		if err != nil {
			panic(err)
		}
		defer zippedFile.Close()
		if _, err := io.Copy(dstFile, zippedFile); err != nil {
			panic(err)
		}
		if ".block" == filepath.Ext(dstFile.Name()) {
			dstPath := path.Join(enablerPath, fmt.Sprintf("channel_genesis.block"))
			transformFile(fileLocation, dstPath)
		} else if ".pem" == filepath.Ext(dstFile.Name()) {
			dstPath := path.Join(enablerPath, fmt.Sprintf("tlsca.example.com-cert.pem"))
			transformFile(fileLocation, dstPath)
		} else {

		}

		// return nil
	}

	//  unzip the file first in a folder and then copy it to the required directory, also check if nothing is missing or not.
}

// Leave is the implemenation of the leave function in the IProvider interface for hyperledger fabric.
// It performs few steps as listed below:
// 1. The leave command is designed to enable an organization leave the network.
// 2. The organization running, updates the network that it wants to leave.
// 3. This organization then creates a envelope file which needs to be sent to other organization for their signatures.
func (f *FabricDefinition) Leave(networkId string, orgName string, userId string, useVolume bool, finalize bool) error {
	userIdentification = userId
	verbose = true
	f.UseVolume = useVolume
	// THe file is generated by the org3 after it has joined the network.
	if !finalize {
		f.leaveNetwork(userId, orgName, networkId)
	} else {
		// also before doing this step, would need to copy the files.
		channelName := fmt.Sprintf("channel%s", strings.ToLower(orgName))
		f.signConfig("config_update_in_envelope_leave.pb")
		f.signAndUpdateConfig("config_update_in_envelope_leave.pb", channelName)
	}

	// The file needs to be signed by the others and uploaded to the

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
	channelName := f.Enabler.Members[0].ChannelName
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", channelName, "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config '/etc/enabler/config.json' > '%s/enabler/config1.json'", f.Enabler.NetworkName, storageType, networkDir))

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return
	}
	return nil
}

// In hyperledger fabric, Anchor peers are the peers with special priveledges:
// 1. These peers have the right to endorse onto transactions
// 2. These peers are discoverable by other peers part of network
// 3. These peers can be thought of as admin peers of the organization.
func (f *FabricDefinition) createAnchorPeer(userID string, networkId string, orgName string) (err error) {
	f.Logger.Printf("Creating anchor peers block for channel")
	var storageType string
	var channelName string
	channelName = fmt.Sprintf("channel%s", strings.ToLower(orgName))
	domainName := "example.com"
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerDirectory := path.Join(constants.EnablerDir, userID, f.Enabler.NetworkName, "enabler")
	networkDir := path.Join(constants.EnablerDir, userID, f.Enabler.NetworkName)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerDirectory
	}
	network := fmt.Sprintf("%s", networkId)
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(f.Enabler.Members[0].OrgName), domainName)
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", fmt.Sprintf("%s", channelName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	_, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", network, storageType, networkDir)).Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq '.channel_group.groups.Application.groups.%sMSP.values += {\"AnchorPeers\":{\"mod_policy\": \"Admins\",\"value\":{\"anchor_peers\": [{\"host\": \"%s\",\"port\": 7051}]},\"version\": \"0\"}}' /etc/enabler/config1.json  > %s/enabler/modified_anchor_config.json ", network, storageType, networkDir, f.Enabler.Members[0].OrgName, peerID, networkDir)).Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/modified_anchor_config.json:/etc/enabler/modified_anchor_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_anchor_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_anchor_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", fmt.Sprintf("%s", channelName), "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_anchor_config.pb", "--output", "/etc/enabler/anchor_update.pb")
	// Required Step
	_, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/anchor_update.pb --type common.ConfigUpdate | jq . > %s/enabler/anchor_update.json", network, storageType, networkDir)).Output()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/anchor_update.json)'}}}'| jq . > %s/enabler/anchor_update_in_envelope.json", network, storageType, channelName, networkDir, networkDir))
	// fmt.Printf("%s", cmd.String())
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/anchor_update_in_envelope.json:/etc/enabler/anchor_update_in_envelope.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/anchor_update_in_envelope.json", "--type", "common.Envelope", "--output", "/etc/enabler/anchor_update_in_envelope.pb")

	// Before doing this need to copy the cafile from the orderer msp-> tlsca .pem to org3 accessible location
	// Then only it would work.
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/tlsca.example.com-cert.pem:/etc/enabler/tlsca.example.com-cert.pem", networkDir), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "update", "-f", "/etc/enabler/anchor_update_in_envelope.pb", "-c", fmt.Sprintf("%s", channelName), "-o", fmt.Sprintf("%s:7050", fmt.Sprintf("fabric_orderer.%s", strings.ToLower(orgName))), "--tls", "--cafile", fmt.Sprintf("%s/tlsca.%s-cert.pem", "/etc/enabler", f.Enabler.Members[0].DomainName))
	// fmt.Printf(" %s\n", out)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "getinfo", "-c", channelName)

}

func (f *FabricDefinition) leaveNetwork(userID string, orgId string, networkId string) (err error) {
	f.Logger.Printf("Leaving the network")
	networkDir := path.Join(constants.EnablerDir, userID, f.Enabler.NetworkName)
	var storageType string
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerPath := path.Join(constants.EnablerDir, userID, f.Enabler.NetworkName, "enabler")
	channelName := fmt.Sprintf("channel%s", strings.ToLower(orgId))
	domainName := "example.com"
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(f.Enabler.Members[0].OrgName), domainName)
	peerID := fmt.Sprintf("%s.%s.%s", "peer0", strings.ToLower(f.Enabler.Members[0].OrgName), domainName)
	network := fmt.Sprintf("%s", networkId)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "config", "/etc/enabler/config_block.pb", "-c", fmt.Sprintf("%s", channelName))
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_decode", "--input", "/etc/enabler/config_block.pb", "--type", "common.Block", "--output", "/etc/enabler/config.json")

	out, err := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq .data.data[0].payload.data.config /etc/enabler/config.json > %s/enabler/config1.json", network, storageType, networkDir)).Output()
	if err != nil {
		return err
	}
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler -v %s/enabler/config1.json:/etc/enabler/config1.json hyperledger/fabric-tools:2.3 jq 'del(.channel_group.groups.Application.groups.%sMSP)' /etc/enabler/config1.json  > %s/enabler/modified_config.json ", network, storageType, networkDir, f.Enabler.Members[0].OrgName, networkDir)).Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", channelName, "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_config.pb", "--output", "/etc/enabler/config_update.pb")
	// Required Step
	out, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/config_update.pb --type common.ConfigUpdate | jq . > %s/enabler/config_update.json", network, storageType, networkDir)).Output()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/config_update.json)'}}}'| jq . > %s/enabler/config_update_in_envelope.json", network, storageType, channelName, networkDir, networkDir))
	fmt.Printf("%s", cmd.String())
	out, err = cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf(" %s\n", out)

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config_update_in_envelope.json:/etc/enabler/config_update_in_envelope.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config_update_in_envelope.json", "--type", "common.Envelope", "--output", "/etc/enabler/config_update_in_envelope_leave.pb")

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "signconfigtx", "-f", fmt.Sprintf("/etc/enabler/%s", "config_update_in_envelope_leave.pb"), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName))

	return nil

}

// This is in the preparation phase
// This function uses the definition file provided by the organization which wishes to join the network and then uses this file to create an envelope, containing the info to join the network.
func (f *FabricDefinition) envelopeBlockCreation(userId string, networkId string, orgName string) (err error) {
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	// Required Step
	var storageType string
	var channelName string
	channelName = f.Enabler.Members[0].ChannelName
	// domainName := "example.com"

	// peerID := fmt.Sprintf("%s.%s.%s", "peer0", strings.ToLower(f.Enabler.Members[0].OrgName), domainName)
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	// orgDefFilePath := path.Join(path.Join(constants.EnablerDir, userId, networkId, "enabler"), fmt.Sprintf("%s.json", orgName))
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerPath
	}
	// Required Step
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 jq -s '.[0] * {\"channel_group\":{\"groups\":{\"Application\":{\"groups\": {\"%sMSP\":.[1]}}}}}' '/etc/enabler/config1.json' '/etc/enabler/%s.json' > '%s/enabler/modified_config.json' ", f.Enabler.NetworkName, storageType, orgName, orgName, networkDir))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/config1.json", "--type", "common.Config", "--output", "/etc/enabler/config1.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", "/etc/enabler/modified_config.json", "--type", "common.Config", "--output", "/etc/enabler/modified_config.pb")
	// Required Step

	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/config1.json:/etc/enabler/config1.json", networkDir), "-v", fmt.Sprintf("%s/enabler/modified_config.json:/etc/enabler/modified_config.json", networkDir), "hyperledger/fabric-tools:2.3",
		"configtxlator", "compute_update", "--channel_id", channelName, "--original", "/etc/enabler/config1.pb", "--updated", "/etc/enabler/modified_config.pb", "--output", fmt.Sprintf("/etc/enabler/%s_update.pb", orgName))
	// Required Step
	_, err = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxlator proto_decode --input /etc/enabler/%s_update.pb --type common.ConfigUpdate | jq . > %s/enabler/%s_update.json", f.Enabler.NetworkName, storageType, orgName, networkDir, orgName)).Output()

	// Required Step
	exec.Command("bash", "-c", fmt.Sprintf("touch %s/enabler/%s_update_in_envelope.json", networkDir, orgName)).Output()

	cmd = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm --network=%s_default -v %s:/etc/enabler hyperledger/fabric-tools:2.3 echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"%s\", \"type\":2}},\"data\":{\"config_update\":'$(cat /%s/enabler/%s_update.json)'}}}'| jq . > %s/enabler/%s_update_in_envelope.json", f.Enabler.NetworkName, storageType, channelName, networkDir, orgName, networkDir, orgName))
	exec.Command("bash", "-c", fmt.Sprintf("touch %s/enabler/%s_update_in_envelope.pb", networkDir, orgName)).Output()
	// fmt.Printf("%s", cmd.String())
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-v", fmt.Sprintf("%s/enabler/%s_update_in_envelope.json:/etc/enabler/%s_update_in_envelope.json", networkDir, orgName, orgName), "hyperledger/fabric-tools:2.3",
		"configtxlator", "proto_encode", "--input", fmt.Sprintf("/etc/enabler/%s_update_in_envelope.json", orgName), "--type", "common.Envelope", "--output", fmt.Sprintf("/etc/enabler/%s_update_in_envelope.pb", orgName))

	return nil
}

func (f *FabricDefinition) signConfig(envelopeFile string) error {
	f.Logger.Printf("Signing the config block for channel")
	filenameWithoutExt := strings.TrimSuffix(envelopeFile, filepath.Ext(envelopeFile))
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

		exec.Command("bash", "-c", fmt.Sprintf("touch %s/enabler/signed_%s.json", networkDir, filenameWithoutExt)).Output()

		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "-v", fmt.Sprintf("%s/enabler/%s:/etc/enabler/%s", networkDir, envelopeFile, envelopeFile), "hyperledger/fabric-tools:2.3",
			"configtxlator", "proto_decode", "--input", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "--type", "common.Envelope", "--output", fmt.Sprintf("/etc/enabler/signed_%s.json", filenameWithoutExt))

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

func (f *FabricDefinition) signAndUpdateConfig(envelopeFile string, channelName string) error {
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	f.Logger.Printf("Sign and Update config block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	// channelName := f.Enabler.Members[0].ChannelName
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", channelName, "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	} else {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", channelName, "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

	}

	return nil
}

func (f *FabricDefinition) signAndUpdateConfigMultipleParticipants(envelopeFile string, channelName string, networkName string, ordererName string, cafile string) error {
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	f.Logger.Printf("Sign and Update config block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	// channelName := f.Enabler.Members[0].ChannelName
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if f.UseVolume {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", networkName), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", channelName, "-o", fmt.Sprintf("%s:7050", ordererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/%s", cafile))
	} else {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", networkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
			fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
			"peer", "channel", "update", "-f", fmt.Sprintf("/etc/enabler/%s", envelopeFile), "-c", channelName, "-o", fmt.Sprintf("%s:7050", ordererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/%s", cafile))

	}

	return nil
}

// This function fetches the gensis block, and then copies the genesis block to the org which requested for it.
// NOTE: The request still needs to be done currently only copy is being done and the genesis block is copied onto the requesting organization's folder/volume.
func (f *FabricDefinition) fetchChannelGenesisBlock() error {
	var storageType string
	var network string
	network = f.Enabler.NetworkName

	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	f.Logger.Printf("Fetching channel genesis block for channel")
	networkDir := path.Join(constants.EnablerDir, userIdentification, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerPath := path.Join(networkDir, "enabler")
	channelName := f.Enabler.Members[0].ChannelName

	cafile := fmt.Sprintf("organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName)

	if f.UseVolume {
		storageType = volumeName

	} else {
		storageType = enablerPath
	}
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3",
		"peer", "channel", "fetch", "0", fmt.Sprintf("/etc/enabler/channel_genesis_block_%s.block", channelName), "-c", channelName, "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--tls", "--cafile", fmt.Sprintf("/etc/enabler/%s", cafile))

	// copy the file channel_genesis_block.block to the another org

	// Here we create the zip file which would be needed for the join, This can be done by creating the zip archive with the genesis block and the cafile together in a zip formaat.

	createZipForAccept(enablerPath, fmt.Sprintf("channel_genesis_block_%s.block", channelName), cafile, fmt.Sprintf("network_config.json"), f.Enabler.Members[0].OrgName)

	// docker.CopyFromContainer(peerID, "/etc/enabler/channel_genesis_block.block", fmt.Sprintf("%s/enabler/channel_genesis_block.block", networkDir), verbose)
	return nil
}

func createZipForInvite(enablerPath string, definitionFile string, networkConfig string, orgName string) {
	archive, err := os.Create(path.Join(enablerPath, fmt.Sprintf("%s_Invite.zip", orgName)))
	if err != nil {
		panic(err)
	}

	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	f1, err := os.Open(path.Join(enablerPath, definitionFile))
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	w1, err := zipWriter.Create(definitionFile)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	f2, err := os.Open(path.Join(enablerPath, networkConfig))
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	w2, err := zipWriter.Create(networkConfig)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w2, f2); err != nil {
		panic(err)
	}
	zipWriter.Close()

}

func createZipForSign(enablerPath string, envelopeFile string, envelopeJson string, networkConfig string, cafile string, orgName string, networkName string) {

	// name should be orgname_accept_transfer.zip
	archive, err := os.Create(path.Join(enablerPath, fmt.Sprintf("%s_sign_transfer.zip", orgName)))
	if err != nil {
		panic(err)
	}

	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	// err = os.Chmod(path.Join(enablerPath, envelopeFile), 0777)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	f1, err := os.Open(path.Join(enablerPath, envelopeFile))
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	w1, err := zipWriter.Create(envelopeFile)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	f2, err := os.Open(path.Join(enablerPath, envelopeJson))
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	w2, err := zipWriter.Create(envelopeJson)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w2, f2); err != nil {
		panic(err)
	}

	f3, err := os.Open(networkConfig)
	if err != nil {
		panic(err)
	}
	defer f3.Close()

	w3, err := zipWriter.Create("network_config.json")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w3, f3); err != nil {
		panic(err)
	}
	f4, err := os.Open(path.Join(enablerPath, cafile))
	if err != nil {
		panic(err)
	}
	defer f4.Close()

	w4, err := zipWriter.Create(fmt.Sprintf("%s_tlsca.example.com-cert.pem", networkName))
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w4, f4); err != nil {
		panic(err)
	}

	zipWriter.Close()

}

func createZipForAccept(enablerPath string, genesisFile string, cafile string, networkConfig string, orgName string) {

	// name should be orgname_accept_transfer.zip
	archive, err := os.Create(path.Join(enablerPath, fmt.Sprintf("%s_accept_transfer.zip", orgName)))
	if err != nil {
		panic(err)
	}

	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	f1, err := os.Open(path.Join(enablerPath, genesisFile))
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	w1, err := zipWriter.Create("channel_genesis.block")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	f2, err := os.Open(path.Join(enablerPath, cafile))
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	w2, err := zipWriter.Create("tlsca.example.com-cert.pem")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w2, f2); err != nil {
		panic(err)
	}

	f3, err := os.Open(path.Join(enablerPath, networkConfig))
	if err != nil {
		panic(err)
	}
	defer f3.Close()

	w3, err := zipWriter.Create(networkConfig)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w3, f3); err != nil {
		panic(err)
	}

	zipWriter.Close()

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
func (f *FabricDefinition) generateCryptoMaterial(userId string, useVolume bool, localSetup bool) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	enablerPath := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	cryptogenYamlPath := path.Join(blockchainDirectory, "cryptogen.yaml")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	orgName := f.Enabler.Members[0].OrgName
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	blockchainDir := path.Join(networkDir, "blockchain")
	var cmd *exec.Cmd

	// volumeName := fmt.Sprintf("enabler_fabric")
	f.Logger.Printf("Generating the volume with volume name: %s", volumeName)
	if err := docker.CreateVolume(volumeName, verbose); err != nil {
		return err
	}
	f.Logger.Printf("Generating the cryptographic certificates for the organization . . .")
	// Run cryptogen to generate MSP
	if !useVolume {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", enablerPath), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
			return err
		}
		cmd = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s/configtx.yaml:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", blockchainDir, enablerDirectory, orgName, enablerDirectory, orgName))

	} else {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenYamlPath), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/etc/enabler/organizations"); err != nil {
			return err
		}
		cmd = exec.Command("bash", "-c", fmt.Sprintf("docker run --rm -v %s/configtx.yaml:/etc/hyperledger/fabric/configtx.yaml -v %s:/etc/enabler hyperledger/fabric-tools:2.3 configtxgen --printOrg %sMSP > %s/%s.json", blockchainDir, volumeName, orgName, enablerDirectory, orgName))

	}
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error occured while creating the definition file", err)
		return err
	}
	f.Logger.Printf("Cryptographic Certificates generated successfully.")
	if err := docker.InspectNetwork(fmt.Sprintf("%s_default", f.Enabler.NetworkName), verbose); err != nil {
		f.Logger.Printf("Creating a docker network . . .")
		if localSetup {
			err = docker.CreateNetwork(fmt.Sprintf("%s_default", f.Enabler.NetworkName), verbose)
			if err != nil {
				fmt.Println("Error occured while creating docker network \n", err)
				return err
			}

		} else {
			err = docker.CreateOverlayNetwork(fmt.Sprintf("%s_default", f.Enabler.NetworkName), verbose)
			if err != nil {
				fmt.Println("Error occured while creating docker swarm network \n", err)
				return err
			}

		}
	}
	f.Logger.Printf("Docker Network Created Successfully !")

	// fmt.Printf("Docker Network created")
	createZipForInvite(enablerPath, fmt.Sprintf("%s.json", orgName), fmt.Sprintf("network_config.json"), orgName)
	// here we need to use the two files and create a zip for them.
	// fmt.Printf("\n\nThe user '%s' has been Successfully initialized. To create the network, run:\n\n go run main.go create -u %s\n\n", userId, userId)

	return nil
}

func (f *FabricDefinition) generateGenesisBlock(userId string, useVolume bool) (err error) {
	blockchainDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain")
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	channelName := f.Enabler.Members[0].ChannelName
	f.Logger.Printf("Generating the Genesis Block for the Blockchain.\n")
	// Generate genesis block
	// might also need to generate the configtx yaml file according the orgname and even the name as example.com does not seem quite good enough
	if !useVolume {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", channelName); err != nil {
			return err
		}
	} else {
		if err := docker.RunDockerCommand(blockchainDirectory, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(blockchainDirectory, "configtx.yaml")), "hyperledger/fabric-tools:2.3", "configtxgen", "-outputBlock", "/etc/enabler/enabler.block", "-profile", "SingleOrgApplicationGenesis", "-channelID", channelName); err != nil {
			return err
		}
	}

	f.Logger.Printf("Generated the Genesis Block successfully.\n")

	return nil
}

func (f *FabricDefinition) createChannel(userId string, useVolume bool) (err error) {
	f.Logger.Printf("Creating channel\n")
	var network string
	network = f.Enabler.NetworkName
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	channelName := f.Enabler.Members[0].ChannelName
	if !useVolume {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "hyperledger/fabric-tools:2.3", "osnadmin", "channel", "join", "--channelID", channelName, "--config-block", "/etc/enabler/enabler.block", "-o", fmt.Sprintf("%s:7053", f.Enabler.Members[0].OrdererName), "--ca-file", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/ca.crt", f.Enabler.Members[0].DomainName), "--client-cert", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.crt", f.Enabler.Members[0].DomainName), "--client-key", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.key", f.Enabler.Members[0].DomainName))

	} else {
		return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "osnadmin", "channel", "join", "--channelID", channelName, "--config-block", "/etc/enabler/enabler.block", "-o", fmt.Sprintf("%s:7053", f.Enabler.Members[0].OrdererName), "--ca-file", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/ca.crt", f.Enabler.Members[0].DomainName), "--client-cert", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.crt", f.Enabler.Members[0].DomainName), "--client-key", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/users/Admin@example.com/tls/client.key", f.Enabler.Members[0].DomainName))

	}
	// volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	// docker.CopyFromContainer(fmt.Sprintf("fabric_orderer"), "/etc/enabler/genesis.block", fmt.Sprintf("%s/enabler/genesis.block", networkDir), verbose)

	// return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", f.Enabler.NetworkName), "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "-v", fmt.Sprintf("%s:/etc/hyperledger/fabric/configtx.yaml", path.Join(path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "blockchain"), "configtx.yaml")), "-e", "CORE_PEER_ADDRESS=peer0.org1.example.com:7051","-e","SYS_CHANNEL=channel1", "-e", "CORE_PEER_TLS_ENABLED=true", "-e", "CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt", "-e", "CORE_PEER_LOCALMSPID=Org1MSP", "-e", "CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp", "hyperledger/fabric-tools:2.3", "peer", "channel", "create", "-c", "channel1", "-f", "/etc/enabler/enabler.tx", "-o", fmt.Sprintf("fabric_orderer:7050"), "--tls", "--cafile", "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem")

}

func (f *FabricDefinition) joinChannel(userId string, useVolume bool) error {
	f.Logger.Printf("Joining channel . . .\n")
	var storageType string
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	var network string
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	orgDomain := fmt.Sprintf("%s.%s", strings.ToLower(f.Enabler.Members[0].OrgName), f.Enabler.Members[0].DomainName)
	// channelName := fmt.Sprintf("channel%s", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerDirectory
	}
	network = f.Enabler.NetworkName
	err := docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/enabler.block")

	if err != nil {
		return err
	}
	// chaincode package
	f.Logger.Printf("Peer joined the channel successfully .\n")
	f.Logger.Printf("Deploying SmartContract.\n")
	err = f.packageAndDeployChaincode(userId, network, storageType, peerID, orgDomain)
	if err != nil {
		return err
	}
	f.Logger.Printf("Smart Contract Installation done ")
	f.Logger.Printf("Smart Contract Deployed Successfully.")
	return nil
	// chaincode approve : if multiple parties are part of the channel, they all need to do this step.
}

func (f *FabricDefinition) packageAndDeployChaincode(userId string, network string, storageType string, peerID string, orgDomain string) (err error) {

	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "lifecycle", "chaincode", "package", "/etc/enabler/mycc.tar.gz", "--path", "/etc/enabler/chaincode/", "--lang", "golang", "--label", "mycc1")
	f.Logger.Printf("Installing SmartContract . . . Hang on this might take a few seconds\n")
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "lifecycle", "chaincode", "install", "/etc/enabler/mycc.tar.gz")

}

func (f *FabricDefinition) queryChaincode(userId string, network string, storageType string, peerID string, orgDomain string) (*QueryInstalledChaincodes, error) {

	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	str, err := docker.RunDockerCommandBuffered(networkDir, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "lifecycle", "chaincode", "queryinstalled")
	if err != nil {
		return nil, err
	}
	var res *QueryInstalledChaincodes
	fmt.Println(str)
	err = json.Unmarshal([]byte(str), &res)
	if err != nil {
		fmt.Println("An error occured unmarshalling", err)
		return nil, err
	}
	return res, nil
}

func (f *FabricDefinition) approveChaincode(packageID string, userId string, network string, storageType string, peerID string, orgDomain string, channelID string) error {

	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain),
		"hyperledger/fabric-tools:2.3", "peer", "lifecycle", "chaincode", "approveformyorg", "-o", fmt.Sprintf("%s:7050", f.Enabler.Members[0].OrdererName), "--ordererTLSHostnameOverride", f.Enabler.Members[0].OrdererName, "--channelID", channelID, "--name", "mycc", "--version", "1.0", "--package-id", packageID, "--sequence", "1", "--tls", "--cafile", fmt.Sprintf("/etc/enabler/organizations/ordererOrganizations/%s/orderers/%s.%s/msp/tlscacerts/tlsca.%s-cert.pem", f.Enabler.Members[0].DomainName, f.Enabler.Members[0].OrdererName, f.Enabler.Members[0].DomainName, f.Enabler.Members[0].DomainName))

}

func (f *FabricDefinition) commitChaincode() {

}

//  the way this function would look like when it is called is
//
// joinOtherOrgPeerToChannel(kinshuk,kinshuk_network1,Org3,"the flag passed would be accept.")
func (f *FabricDefinition) joinOtherOrgPeerToChannel(userId string, networkId string, orgName string) error {
	f.Logger.Printf("Joining other peers to the new channel")
	// Note : Need to be changed since we need here channel org1 instead of channel Org3 since this part of code is run on different machine running Org3, N/w3 this channelName needs to be different.
	channelName := fmt.Sprintf("channel%s", strings.ToLower(orgName))
	// also in this volume need the network id for the Org3 instead of Network id for Org1
	var network string
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	var storageType string
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	if f.UseVolume {
		storageType = volumeName
	} else {
		storageType = enablerDirectory
	}
	network = fmt.Sprintf("%s", networkId)
	orgDomain := fmt.Sprintf("%s.example.com", strings.ToLower(f.Enabler.Members[0].OrgName))
	peerID := fmt.Sprintf("%s.%s", f.Enabler.Members[0].NodeName, orgDomain)
	// also pass external network here too. since the default network would be different .
	docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "join", "-b", "/etc/enabler/channel_genesis.block")
	return docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", fmt.Sprintf("--network=%s_default", network), "-v", fmt.Sprintf("%s:/etc/enabler", storageType), "-e", fmt.Sprintf("CORE_PEER_ADDRESS=%s:7051", peerID), "-e", "CORE_PEER_TLS_ENABLED=true", "-e", fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID), "-e", fmt.Sprintf("CORE_PEER_LOCALMSPID=%sMSP", f.Enabler.Members[0].OrgName), "-e", fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain), "hyperledger/fabric-tools:2.3", "peer", "channel", "getinfo", "-c", fmt.Sprintf("%s", channelName))
}

func (f *FabricDefinition) getBlockInformation(userId string, useVolume bool) error {
	// f.Logger.Printf("Get block information")
	enablerDirectory := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName, "enabler")
	networkDir := path.Join(constants.EnablerDir, userId, f.Enabler.NetworkName)
	volumeName := fmt.Sprintf("%s_fabric", f.Enabler.NetworkName)
	if useVolume {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", volumeName), "hyperledger/fabric-tools:2.3", "configtxlator", "proto_decode", "--input", "/etc/enabler/enabler.block", "--output", "/etc/enabler/enabler.json", "--type", "common.Block")

	} else {
		docker.RunDockerCommand(networkDir, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/enabler", enablerDirectory), "hyperledger/fabric-tools:2.3", "configtxlator", "proto_decode", "--input", "/etc/enabler/enabler.block", "--output", "/etc/enabler/enabler.json", "--type", "common.Block")
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
