/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"BlockchainEnabler/BlockchainEnabler/internal/deployer/docker"
	"BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"
	"fmt"

	"github.com/spf13/cobra"
)

var networkId string
var userId string
var createPlatformManager *enablerplatform.EnablerPlatformManager
var useSDK bool
var basic bool

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "This command is for setting up and running the network",
	Long: `Some steps involve while running this network are:
	1. Checking if the docker is present in the host machine or not.
	2. Loading the enabler information -> regarding the members.
	3. Creating the gensis block and the configuration files.
	4. Running the containers for the whole -> orderer, ca, peer and other if needed.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("create called")
		if err := docker.CheckDockerConfig(); err != nil {
			return err
		}
		createPlatformManager = enablerplatform.GetInstance(&logger)
		// steps to follow the user specifies the name of the platform and then we run its containers.
		// it needs to load in the basic file from the directory and initialize it with the values for the network.
		// We need to then check which kind of network it is and then we would call the network functions(objects).
		createPlatformManager.LoadUser(networkId, userId)
		logger.Printf(createPlatformManager.UserId)
		fmt.Printf("The value of sdk is %v", useSDK)
		if useSDK {
			createPlatformManager.CreateNetworkUsingSDK(useVolume, basic)
		} else {
			createPlatformManager.CreateNetwork(useVolume, basic)
		}

		// one more thing to consider is to before running the network actually checking if the ports are available or not and then if not then changing the ports and
		// making these changes to the file generated-> docker compose as well as the others. Regarding the port information -> this can be in back log.
		// Also need to figure how to append the file that is generated inside the folder and then using this file to track all the parameters.
		// 1 network can also have multiple members currently we are considering only 1 member. But the members can be multiple for the network.
		// Now the user can have with the same user id created multiple
		// Now we need to load the network information from the directory in order to set the corresponding values.
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&userId, "userId", "u", "", "Provide the user Id for the network you want to run.")
	createCmd.Flags().StringVarP(&networkId, "netid", "n", "", "Provide the network id of the network you want to run.")
	createCmd.Flags().BoolVarP(&useSDK, "useSDK", "l", false, "Function to enable or disable the use of SDK default: false")
	createCmd.Flags().BoolVarP(&basic, "simpleSetup", "s", false, "Function to enable or disable the use of Basic setup default: false")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
