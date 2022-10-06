// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fabric

import (
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Signature struct {
	Sign       string          `json:"signature,omitempty"`
	SignHeader SignatureHeader `json:"signature_header,omitempty"`
}

type SignatureCreator struct {
	ID    string `json:"id_bytes,omitempty"`
	MspID string `json:"mspid,omitempty"`
}
type SignatureHeader struct {
	Creator SignatureCreator `json:"creator,omitempty"`
	Nonce   string           `json:"nonce,omitempty"`
}

type Registrar struct {
	EnrollID     string `yaml:"enrollId,omitempty"`
	EnrollSecret string `yaml:"enrollSecret,omitempty"`
}

type Path struct {
	Path string `yaml:"path,omitempty"`
}

type NetworkEntity struct {
	TLSCACerts *Path      `yaml:"tlsCACerts,omitempty"`
	URL        string     `yaml:"url,omitempty"`
	Registrar  *Registrar `yaml:"registrar,omitempty"`
}

type ChannelPeer struct {
	ChaincodeQuery bool `yaml:"chaincodeQuery,omitempty"`
	EndorsingPeer  bool `yaml:"endorsingPeer,omitempty"`
	EventSource    bool `yaml:"eventSource,omitempty"`
	LedgerQuery    bool `yaml:"ledgerQuery,omitempty"`
}

type Channel struct {
	Orderers []string                `yaml:"orderers,omitempty"`
	Peers    map[string]*ChannelPeer `yaml:"peers,omitempty"`
}

type Provider struct {
	Provider string `yaml:"provider,omitempty"`
}

type BCCSPSecurity struct {
	Default       *Provider `yaml:"default,omitempty"`
	Enabled       bool      `yaml:"enabled,omitempty"`
	HashAlgorithm string    `yaml:"hashAlgorithm,omitempty"`
	Level         int       `yaml:"level,omitempty"`
	SoftVerify    bool      `yaml:"softVerify,omitempty"`
}

type BCCSP struct {
	Security *BCCSPSecurity `yaml:"security,omitempty"`
}

type CredentialStore struct {
	CryptoStore *Path  `yaml:"cryptoStore,omitempty"`
	Path        string `yaml:"path,omitempty"`
}

type Logging struct {
	Level string `yaml:"level,omitempty"`
}

type TLSCertsClient struct {
	Cert *Path `yaml:"cert,omitempty"`
	Key  *Path `yaml:"key,omitempty"`
}

type TLSCerts struct {
	Client *TLSCertsClient `yaml:"client,omitempty"`
}

type Organization struct {
	CertificateAuthorities []string `yaml:"certificateAuthorities,omitempty"`
	CryptoPath             string   `yaml:"cryptoPath,omitempty"`
	MSPID                  string   `yaml:"mspid,omitempty"`
	Peers                  []string `yaml:"peers,omitempty"`
}

type Client struct {
	BCCSP           *BCCSP           `yaml:"BCCSP,omitempty"`
	CredentialStore *CredentialStore `yaml:"credentialStore"`
	CryptoConfig    *Path            `yaml:"cryptoconfig,omitempty"`
	Logging         *Logging         `yaml:"logging,omitempty"`
	Organization    string           `yaml:"organization,omitempty"`
	TLSCerts        *TLSCerts        `yaml:"tlsCerts,omitempty"`
}

type FabricNetworkConfig struct {
	CertificateAuthorities map[string]*NetworkEntity `yaml:"certificateAuthorities,omitempty"`
	Channels               map[string]*Channel       `yaml:"channels,omitempty"`
	Client                 *Client                   `yaml:"client,omitempty"`
	Organization           string                    `yaml:"organization,omitempty"`
	Orderers               map[string]*NetworkEntity `yaml:"orderers,omitempty"`
	Organizations          map[string]*Organization  `yaml:"organizations,omitempty"`
	Peers                  map[string]*NetworkEntity `yaml:"peers,omitempty"`
	Version                string                    `yaml:"version,omitempty"`
}

func WriteNetworkConfig(outputPath string, enablerPath string, member types.Member) error {
	var peerName string
	var orgDomain string
	domainName := "example.com"
	orgDomain = fmt.Sprintf("%s.%s", strings.ToLower(member.OrgName), domainName)
	peerName = fmt.Sprintf("%s.%s", member.NodeName, orgDomain)

	networkConfig := &FabricNetworkConfig{
		CertificateAuthorities: map[string]*NetworkEntity{
			fmt.Sprintf("%s", orgDomain): {
				TLSCACerts: &Path{
					Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/ca/fabric_ca.%s-cert.pem", enablerPath, orgDomain, orgDomain),
				},
				URL: "http://fabric_ca:7054",
				Registrar: &Registrar{
					EnrollID:     "admin",
					EnrollSecret: "adminpw",
				},
			},
		},
		Channels: map[string]*Channel{
			"enablerchannel": {
				Orderers: []string{fmt.Sprintf("%s", member.OrdererName)},
				Peers: map[string]*ChannelPeer{
					fmt.Sprintf("%s", peerName): {
						ChaincodeQuery: true,
						EndorsingPeer:  true,
						EventSource:    true,
						LedgerQuery:    true,
					},
				},
			},
		},
		Client: &Client{
			BCCSP: &BCCSP{
				Security: &BCCSPSecurity{
					Default: &Provider{
						Provider: "SW",
					},
					Enabled:       true,
					HashAlgorithm: "SHA2",
					Level:         256,
					SoftVerify:    true,
				},
			},
			CredentialStore: &CredentialStore{
				CryptoStore: &Path{
					Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/msp", enablerPath, orgDomain),
				},
				Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/msp", enablerPath, orgDomain),
			},
			CryptoConfig: &Path{
				Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/msp", enablerPath, orgDomain),
			},
			Logging: &Logging{
				Level: "info",
			},
			Organization: fmt.Sprintf("%s", orgDomain),
			TLSCerts: &TLSCerts{
				Client: &TLSCertsClient{
					Cert: &Path{
						Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/users/Admin@%s/tls/client.crt", enablerPath, orgDomain, orgDomain),
					},
					Key: &Path{
						Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/users/Admin@%s/tls/client.key", enablerPath, orgDomain, orgDomain),
					},
				},
			},
		},
		Orderers: map[string]*NetworkEntity{
			fmt.Sprintf("%s", member.OrdererName): {
				TLSCACerts: &Path{
					Path: fmt.Sprintf("%s/organizations/ordererOrganizations/%s/tlsca/tlsca.%s-cert.pem", enablerPath, domainName, domainName),
				},
				URL: fmt.Sprintf("grpcs://%s:7050", member.OrdererName),
			},
		},
		Organizations: map[string]*Organization{
			fmt.Sprintf("%s", orgDomain): {
				CertificateAuthorities: []string{fmt.Sprintf("%s", orgDomain)},
				CryptoPath:             fmt.Sprintf("%s/organizations/peerOrganizations/%s/users/Admin@%s/msp", enablerPath, orgDomain, orgDomain),
				MSPID:                  fmt.Sprintf("%sMSP", member.OrgName),
				Peers:                  []string{peerName},
			},
		},
		Peers: map[string]*NetworkEntity{
			peerName: {
				TLSCACerts: &Path{
					Path: fmt.Sprintf("%s/organizations/peerOrganizations/%s/tlsca/tlsfabric_ca.%s-cert.pem", enablerPath, orgDomain, orgDomain),
				},
				URL: fmt.Sprintf("grpcs://%s:7051", peerName),
			},
		},
		Version: "1.1.0%",
	}
	networkConfigBytes, _ := yaml.Marshal(networkConfig)
	return ioutil.WriteFile(outputPath, networkConfigBytes, 0755)
}
