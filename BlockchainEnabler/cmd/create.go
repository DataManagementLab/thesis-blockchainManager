/*
Copyright © 2022 Kinshuk Kislay  <kinshuk.kislay@stud.tu-darmstadt.de>

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
		fmt.Println("Running create command")
		if err := docker.CheckDockerConfig(); err != nil {
			return err
		}
		createPlatformManager = enablerplatform.GetInstance(&logger)
		// Loads the network configuration for the initilized user.
		if err := createPlatformManager.LoadUser(networkId, userId); err != nil {
			return err
		}
		// Creates the network for the given User
		if err := createPlatformManager.CreateNetwork(useVolume,userLogging); err != nil {
			return err
		}

		// one more thing to consider is to before running the network actually checking if the ports are available or not and then if not then changing the ports and
		// making these changes to the file generated-> docker compose as well as the others. Regarding the port information -> this can be in back log.
		// Also need to figure how to append the file that is generated inside the folder and then using this file to track all the parameters.
		// 1 network can also have multiple members currently we are considering only 1 member. But the members can be multiple for the network.
		// Now the user can have with the same user id created multiple
		// Now we need to load the network information from the directory in order to set the corresponding values.
		
		// Provides the message to the user once the network is successfully created.
		fmt.Printf("\n\nThe Network '%s' for user '%s' has been Successfully created.\n\n", createPlatformManager.Enablers[0].NetworkName, userId)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&userId, "userId", "u", "", "Provide the user Id for the network you want to run.")
	createCmd.Flags().StringVarP(&networkId, "netid", "n", "", "Provide the network id of the network you want to run.")
}
