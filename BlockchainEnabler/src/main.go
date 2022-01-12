package main

import (
	"BlockchainEnabler/BlockchainEnabler/src/provider/fabric"
	// "fmt"
)

func main()  {
	path := "/Users/kinshukkislay/Project/hyperledger/BlockchainEnabler/BlockchainEnabler/configurations/config.yaml"
	fab := fabric.Fabric{
		ChannelID: "mychannel",
	}
	fabric.Initialze(&fab,path)
}