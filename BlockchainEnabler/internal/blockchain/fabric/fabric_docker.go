package fabric

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"

	"gopkg.in/yaml.v2"
)

type FabricDocker struct{}

// either need to handle the ports issue here as the enabler_external port takes an interaface, it would also be easy to just assign the ports here.

func GenerateServiceDefinitions(member *types.Member, memberId string, useVolume bool, userID string) ([]*docker.ServiceDefinition, error) {
	external, ok := member.ExternalPorts.(map[string]int)

	var fileDirectory string
	var orgDomain string
	var peerID string
	orgDomain = fmt.Sprintf("%s.example.com", strings.ToLower(member.OrgName))
	peerID = fmt.Sprintf("%s.%s", member.NodeName, orgDomain)

	if !ok {

	}
	if useVolume {
		fileDirectory = fmt.Sprintf("%s:/etc/enabler", "fabric")
	} else {
		fileDirectory = fmt.Sprintf("%s:/etc/enabler", path.Join(constants.EnablerDir, userID, memberId, "enabler"))
	}
	// dockerNetwork := docker.DockerNetwork{DockerNetworkName: memberId}
	serviceDefinitions := []*docker.ServiceDefinition{
		// Fabric CA
		{
			ServiceName: fmt.Sprintf("fabric_ca"),
			Service: &docker.Service{
				Image:         "hyperledger/fabric-ca:1.5",
				ContainerName: fmt.Sprintf("fabric_ca"),
				Environment: map[string]string{
					"FABRIC_CA_HOME":                            "/etc/hyperledger/fabric-ca-server",
					"FABRIC_CA_SERVER_CA_NAME":                  "fabric_ca",
					"FABRIC_CA_SERVER_PORT":                     fmt.Sprintf("%d", external["ca_server_port"]),
					"FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS": fmt.Sprintf("0.0.0.0:%d", external["ca_operations_listen_port"]),
					"FABRIC_CA_SERVER_CA_CERTFILE":              fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/ca/fabric_ca.%s-cert.pem", orgDomain, orgDomain),
					"FABRIC_CA_SERVER_CA_KEYFILE":               fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/ca/priv_sk", orgDomain),
				},
				Ports: []string{
					fmt.Sprintf("%d:%d", external["ca_server_port"], external["ca_server_port"]),
					fmt.Sprintf("%d:%d", external["ca_operations_listen_port"], external["ca_operations_listen_port"]),
				},
				Command: "sh -c 'fabric-ca-server start -b admin:adminpw'",
				Volumes: []string{
					fileDirectory,
				},
				DockerNetworkNames: []string{
					"byfn",
				},
			},
			VolumeNames: []string{"fabric_ca", "fabric"},
		},

		// Fabric Orderer
		{
			ServiceName: fmt.Sprintf("fabric_orderer"),
			Service: &docker.Service{
				Image:         "hyperledger/fabric-orderer:2.3",
				ContainerName: fmt.Sprintf("fabric_orderer"),
				Environment: map[string]string{
					"FABRIC_LOGGING_SPEC":             "INFO",
					"ORDERER_GENERAL_LISTENADDRESS":   "0.0.0.0",
					"ORDERER_GENERAL_LISTENPORT":      fmt.Sprint(external["orderer_general_listen_port"]),
					"ORDERER_GENERAL_LOCALMSPID":      "OrdererMSP",
					"ORDERER_GENERAL_LOCALMSPDIR":     "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp",
					"ORDERER_GENERAL_TLS_ENABLED":     "true",
					"ORDERER_GENERAL_TLS_PRIVATEKEY":  "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_GENERAL_TLS_CERTIFICATE": "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_GENERAL_TLS_ROOTCAS":     "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt",
					// "ORDERER_GENERAL_GENESISMETHOD":             "file",
					// "ORDERER_GENERAL_GENESISFILE":               "/etc/enabler/genesis.block",
					"ORDERER_KAFKA_TOPIC_REPLICATIONFACTOR":     "1",
					"ORDERER_KAFKA_VERBOSE":                     "true",
					"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE": "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY":  "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_GENERAL_CLUSTER_ROOTCAS":           "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt",
					"ORDERER_GENERAL_BOOTSTRAPMETHOD":           "none",
					"ORDERER_CHANNELPARTICIPATION_ENABLED":      "true",
					"ORDERER_ADMIN_TLS_ENABLED":                 "true",
					"ORDERER_ADMIN_TLS_CERTIFICATE":             "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_ADMIN_TLS_PRIVATEKEY":              "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_ADMIN_TLS_ROOTCAS":                 "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt",
					"ORDERER_ADMIN_TLS_CLIENTROOTCAS":           "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt",
					"ORDERER_ADMIN_LISTENADDRESS":               fmt.Sprintf("0.0.0.0:%d", external["orderer_admin_listen_port"]),
					"ORDERER_OPERATIONS_LISTENADDRESS":          fmt.Sprintf("0.0.0.0:%d", external["orderer_operations_listen_port"]),
				},
				WorkingDir: "/opt/gopath/src/github.com/hyperledger/fabric",
				Command:    "orderer",
				Volumes: []string{
					fileDirectory,
					fmt.Sprintf("fabric_orderer:/var/hyperledger/production/orderer"),
				},
				Ports: []string{
					fmt.Sprintf("%d:%d", external["orderer_general_listen_port"], external["orderer_general_listen_port"]),
					fmt.Sprintf("%d:%d", external["orderer_admin_listen_port"], external["orderer_admin_listen_port"]),
					fmt.Sprintf("%d:%d", external["orderer_operations_listen_port"], external["orderer_operations_listen_port"]),
				},
				DockerNetworkNames: []string{
					"byfn",
				},
			},
			VolumeNames: []string{"fabric_orderer"},
		},

		// Fabric Peer
		{
			ServiceName: fmt.Sprintf("%s", peerID),
			Service: &docker.Service{
				Image:         "hyperledger/fabric-peer:2.3",
				ContainerName: fmt.Sprintf(peerID),
				Command:       "peer node start",
				Environment: map[string]string{
					"CORE_VM_ENDPOINT":                      "unix:///host/var/run/docker.sock",
					"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE": fmt.Sprintf("%s_default", memberId),
					"FABRIC_LOGGING_SPEC":                   "INFO",
					"CORE_PEER_TLS_ENABLED":                 "true",
					"CORE_PEER_PROFILE_ENABLED":             "false",
					"CORE_PEER_MSPCONFIGPATH":               fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/users/Admin@%s/msp", orgDomain, orgDomain),
					"CORE_PEER_TLS_CERT_FILE":               fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/server.crt", orgDomain, peerID),
					"CORE_PEER_TLS_KEY_FILE":                fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/server.key", orgDomain, peerID),
					"CORE_PEER_TLS_ROOTCERT_FILE":           fmt.Sprintf("/etc/enabler/organizations/peerOrganizations/%s/peers/%s/tls/ca.crt", orgDomain, peerID),
					"CORE_PEER_ID":                          fmt.Sprintf("%s", peerID),
					"CORE_PEER_ADDRESS":                     fmt.Sprintf("%s:%d", peerID, external["core_peer_listen_address_gossip_port"]),
					"CORE_PEER_LISTENADDRESS":               fmt.Sprintf("0.0.0.0:%d", external["core_peer_listen_address_gossip_port"]),
					"CORE_PEER_CHAINCODEADDRESS":            fmt.Sprintf("%s:%d", peerID, external["core_peer_chaincode_listen_port"]),
					"CORE_PEER_CHAINCODELISTENADDRESS":      fmt.Sprintf("0.0.0.0:%d", external["core_peer_chaincode_listen_port"]),
					"CORE_PEER_GOSSIP_BOOTSTRAP":            fmt.Sprintf("%s:%d", peerID, external["core_peer_listen_address_gossip_port"]),
					"CORE_PEER_GOSSIP_EXTERNALENDPOINT":     fmt.Sprintf("%s:%d", peerID, external["core_peer_listen_address_gossip_port"]),
					"CORE_PEER_LOCALMSPID":                  fmt.Sprintf("%sMSP", member.OrgName),
					"CORE_OPERATIONS_LISTENADDRESS":         fmt.Sprintf("0.0.0.0:%d", external["core_operations_listen_port"]),
				},
				Volumes: []string{
					fileDirectory,
					fmt.Sprintf("%s:/var/hyperledger/production", peerID),
					"/var/run/docker.sock:/host/var/run/docker.sock",
				},
				Ports: []string{
					fmt.Sprintf("%d:%d", external["core_peer_listen_address_gossip_port"], external["core_peer_listen_address_gossip_port"]),
					fmt.Sprintf("%d:%d", external["core_operations_listen_port"], external["core_operations_listen_port"]),
				},
				DockerNetworkNames: []string{
					"byfn",
				},
			},
			VolumeNames: []string{fmt.Sprintf("%s", peerID)},
		},
	}
	return serviceDefinitions, nil
}

func (fabDocker *FabricDocker) Deploy(workingDir string) error {
	// Needs to now run the docker containers this can be done using the docker compose file
	// fmt.Printf("Working Directory for docker %s", workingDir)

	err := docker.RunDockerComposeCommand(workingDir, true, true, "up", "-d")
	if err != nil {
		return err
	}
	return nil
}

func (fabDocker *FabricDocker) GenerateFiles(enabler *types.Network, userId string, useVolume bool) (err error) {
	fmt.Printf("The value of the user id %s", userId)
	// dockerNetwor := docker.DockerNetwork {
	// 	DockerNetName: },
	// }
	// dockerExternalNetwork:= docker.externdocker.DockerNetworkExternal{

	// }

	// dockerNetName := docker.DockerNetworkName{
	// 	DockerExternalNetworkName: enabler.NetworkName,
	// }

	dockerNet := docker.DockerNetwork{
		DockerExternalNetwork: &docker.DockerNetworkName{DockerExternalNetworkName: fmt.Sprintf("%s_default", enabler.NetworkName)},
	}

	compose := docker.CreateDockerCompose()
	for _, member := range enabler.Members {
		serviceDefinition, err := GenerateServiceDefinitions(member, fmt.Sprintf("%s", enabler.NetworkName), useVolume, userId)
		if err != nil {
			return err
		}
		for _, services := range serviceDefinition {
			compose.Services[services.ServiceName] = services.Service
			compose.Networks["byfn"] = &dockerNet
			for _, volumeName := range services.VolumeNames {
				compose.Volumes[volumeName] = struct{}{}
			}
		}
		if err := writeDockerCompose(compose, enabler, userId); err != nil {
			return err
		}
	}

	// now need to check for the docker service definition and how to create it .
	// return GenerateServiceDefinitions(enablerName)
	return nil
}

func writeDockerCompose(compose *docker.DockerComposeConfig, enabler *types.Network, userId string) error {
	bytes, err := yaml.Marshal(compose)
	if err != nil {
		return err
	}

	enablerDir := filepath.Join(constants.EnablerDir, userId, enabler.NetworkName)

	return ioutil.WriteFile(filepath.Join(enablerDir, "docker-compose.yml"), bytes, 0755)
}
func GetFabricDockerInstance() *FabricDocker {
	return &FabricDocker{}
}
