package fabric

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"BlockchainEnabler/BlockchainEnabler/internal/constants"
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/types"

	"gopkg.in/yaml.v2"
)

type FabricDocker struct{}

// either need to handle the ports issue here as the enabler_external port takes an interaface, it would also be easy to just assign the ports here.

func GenerateServiceDefinitions(enabler *types.Network) []*docker.ServiceDefinition {
	serviceDefinitions := []*docker.ServiceDefinition{
		// Fabric CA
		{
			ServiceName: "fabric_ca",
			Service: &docker.Service{
				Image:         "hyperledger/fabric-ca:1.5",
				ContainerName: fmt.Sprintf("%s_fabric_ca", enabler.NetworkName),
				Environment: map[string]string{
					"FABRIC_CA_HOME":                            "/etc/hyperledger/fabric-ca-server",
					"FABRIC_CA_SERVER_CA_NAME":                  "fabric_ca",
					"FABRIC_CA_SERVER_PORT":                     "7054",
					"FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS": "0.0.0.0:17054",
					"FABRIC_CA_SERVER_CA_CERTFILE":              "/etc/enabler/organizations/peerOrganizations/org1.example.com/ca/fabric_ca.org1.example.com-cert.pem",
					"FABRIC_CA_SERVER_CA_KEYFILE":               "/etc/enabler/organizations/peerOrganizations/org1.example.com/ca/priv_sk",
				},
				Ports: []string{
					"7054:7054",
					"17054:17054",
				},
				Command: "sh -c 'fabric-ca-server start -b admin:adminpw'",
				Volumes: []string{
					"enabler_fabric:/etc/enabler",
				},
			},
			VolumeNames: []string{"fabric_ca"},
		},

		// Fabric Orderer
		{
			ServiceName: "fabric_orderer",
			Service: &docker.Service{
				Image:         "hyperledger/fabric-orderer:2.3",
				ContainerName: fmt.Sprintf("%s_fabric_orderer", enabler.NetworkName),
				Environment: map[string]string{
					"FABRIC_LOGGING_SPEC":                       "INFO",
					"ORDERER_GENERAL_LISTENADDRESS":             "0.0.0.0",
					"ORDERER_GENERAL_LISTENPORT":                "7050",
					"ORDERER_GENERAL_LOCALMSPID":                "OrdererMSP",
					"ORDERER_GENERAL_LOCALMSPDIR":               "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/msp",
					"ORDERER_GENERAL_TLS_ENABLED":               "true",
					"ORDERER_GENERAL_TLS_PRIVATEKEY":            "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_GENERAL_TLS_CERTIFICATE":           "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_GENERAL_TLS_ROOTCAS":               "[/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt]",
					"ORDERER_KAFKA_TOPIC_REPLICATIONFACTOR":     "1",
					"ORDERER_KAFKA_VERBOSE":                     "true",
					"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE": "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY":  "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_GENERAL_CLUSTER_ROOTCAS":           "[/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt]",
					"ORDERER_GENERAL_BOOTSTRAPMETHOD":           "none",
					"ORDERER_CHANNELPARTICIPATION_ENABLED":      "true",
					"ORDERER_ADMIN_TLS_ENABLED":                 "true",
					"ORDERER_ADMIN_TLS_CERTIFICATE":             "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.crt",
					"ORDERER_ADMIN_TLS_PRIVATEKEY":              "/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/server.key",
					"ORDERER_ADMIN_TLS_ROOTCAS":                 "[/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt]",
					"ORDERER_ADMIN_TLS_CLIENTROOTCAS":           "[/etc/enabler/organizations/ordererOrganizations/example.com/orderers/fabric_orderer.example.com/tls/ca.crt]",
					"ORDERER_ADMIN_LISTENADDRESS":               "0.0.0.0:7053",
					"ORDERER_OPERATIONS_LISTENADDRESS":          "0.0.0.0:17050",
				},
				WorkingDir: "/opt/gopath/src/github.com/hyperledger/fabric",
				Command:    "orderer",
				Volumes: []string{
					"enabler_fabric:/etc/enabler",
					"fabric_orderer:/var/hyperledger/production/orderer",
				},
				Ports: []string{
					"7050:7050",
					"7053:7053",
					"17050:17050",
				},
			},
			VolumeNames: []string{"fabric_orderer"},
		},

		// Fabric Peer
		{
			ServiceName: "fabric_peer",
			Service: &docker.Service{
				Image:         "hyperledger/fabric-peer:2.3",
				ContainerName: fmt.Sprintf("%s_fabric_peer", enabler.NetworkName),
				Environment: map[string]string{
					"CORE_VM_ENDPOINT":                      "unix:///host/var/run/docker.sock",
					"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE": fmt.Sprintf("%s_default", enabler.NetworkName),
					"FABRIC_LOGGING_SPEC":                   "INFO",
					"CORE_PEER_TLS_ENABLED":                 "true",
					"CORE_PEER_PROFILE_ENABLED":             "false",
					"CORE_PEER_MSPCONFIGPATH":               "/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/msp",
					"CORE_PEER_TLS_CERT_FILE":               "/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/server.crt",
					"CORE_PEER_TLS_KEY_FILE":                "/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/server.key",
					"CORE_PEER_TLS_ROOTCERT_FILE":           "/etc/enabler/organizations/peerOrganizations/org1.example.com/peers/fabric_peer.org1.example.com/tls/ca.crt",
					"CORE_PEER_ID":                          "fabric_peer",
					"CORE_PEER_ADDRESS":                     "fabric_peer:7051",
					"CORE_PEER_LISTENADDRESS":               "0.0.0.0:7051",
					"CORE_PEER_CHAINCODEADDRESS":            "fabric_peer:7052",
					"CORE_PEER_CHAINCODELISTENADDRESS":      "0.0.0.0:7052",
					"CORE_PEER_GOSSIP_BOOTSTRAP":            "fabric_peer:7051",
					"CORE_PEER_GOSSIP_EXTERNALENDPOINT":     "fabric_peer:7051",
					"CORE_PEER_LOCALMSPID":                  "Org1MSP",
					"CORE_OPERATIONS_LISTENADDRESS":         "0.0.0.0:17051",
				},
				Volumes: []string{
					"enabler_fabric:/etc/enabler",
					"fabric_peer:/var/hyperledger/production",
					"/var/run/docker.sock:/host/var/run/docker.sock",
				},
				Ports: []string{
					"7051:7051",
					"17051:17051",
				},
			},
			VolumeNames: []string{"fabric_peer"},
		},
	}
	return serviceDefinitions
}

func (fabDocker *FabricDocker) Deploy() {

}

func (fabDocker *FabricDocker) GenerateFiles(enabler *types.Network, userId string) (err error) {
	compose := docker.CreateDockerCompose()
	serviceDefinition := GenerateServiceDefinitions(enabler)
	for _, services := range serviceDefinition {
		compose.Services[services.ServiceName] = services.Service
		for _, volumeName := range services.VolumeNames {
			compose.Volumes[volumeName] = struct{}{}
		}
	}
	if err := writeDockerCompose(compose, enabler, userId); err != nil {
		return err
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
