package conf

import (
	_ "fmt"

	"BlockchainEnabler/BlockchainEnabler/internal/types"
)

type InitializationOptions struct {
	NumberOfMembers   int
	UserId            string
	ConfigurationFile string
	BlockchainType    types.BlockchainProvider
	NetworkName       string
	OrgNames          []string
	NodeNames         []string
	ServicesPort      int //only if the user specifies a specific services port he wants to utilize.
	UseVolume         bool
	BasicSetup        bool
	UserLogging       bool
}
