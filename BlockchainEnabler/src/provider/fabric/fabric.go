package fabric

import (
	// "fmt"
	// "github.com/hyperledger/fabric-sdk-go"

	"fmt"
	"github.com/pkg/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	
	// "include"
)

type Fabric struct{
	ChannelID string
	sdk *fabsdk.FabricSDK
}

func Initialze(fabric *Fabric, configPath string)  {
	// configPath := "../../../configurations/configuration.yaml"
	err := runSDK(config.FromFile(configPath),fabric)
	if err !=nil{
		fmt.Printf("%v",err.Error())
	}
}

func runSDK(configProvider core.ConfigProvider, fabric *Fabric) error{
	sdk, err := fabsdk.New(configProvider)
	if err != nil{
		return errors.WithMessage(err, "failed to create SDK")
	}
	fabric.sdk = sdk
	
	return nil
}
