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
	"BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"
	"fmt"

	"github.com/spf13/cobra"
)

var joinPlatformManager *enablerplatform.EnablerPlatformManager
var zipFile string
var basic bool

// var finalize bool

// joinCmd represents the join command
var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join command adds another  organization to the created network.",
	Long: `Join command is run by the organization which wants to join the network hosted and created by another organization.
	The join command performs the following steps:
	1. Uses the zip file containing the network configuration to to join the network.
	2. Updates the network to be joined by this new organization.
	3. Adds the peers belonging to the organization running join, to become part of the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running join command")
		// since we now have the name of the org we want to create, first step would be to generate the org file, and then create the definition file for the org.
		// Files currently needed
		// 1. crypto file to generate the cryptographic files for the org
		// 2. Generate the configtx.yaml file which defines the configurations
		// 3. Docker file to run the container for the peer.

		// Now when the user specifies the user id and the network, the network related configurations should be loaded,
		// After they are loaded, then we initialize the instance as a fabric / ethereum instance which could be then implemented.

		joinPlatformManager = enablerplatform.GetInstance(&logger)
		if err := joinPlatformManager.LoadUser("", userId); err != nil {
			return err
		}
		// logger.Printf(invitePlatformManager.UserId)
		if err := joinPlatformManager.JoinNetwork(useVolume, zipFile, basic,userLogging); err != nil {
			return err
		}
		return nil

	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVarP(&userId, "userId", "u", "", "The User ID for the user.")
	joinCmd.Flags().StringVarP(&zipFile, "zipFile", "z", "", "The zip of the files needed.")
	joinCmd.Flags().BoolVarP(&basic, "simpleSetup", "s", false, "Function to enable or disable the use of Basic setup default: false")
}
