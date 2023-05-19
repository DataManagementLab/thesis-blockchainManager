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

var invitePlatformManager *enablerplatform.EnablerPlatformManager
var file string
var ordererInfo string

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add command is used to add another organization to the network.",
	Long: `The add is executed by the actor when it wants to add another organization to the created network.
	1. Add is run by organization which invites other org to join its network.
	2. It prepares the network to be joined by other organization.
	3. Once successfully executed, the invited organization can use the join command to join the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running add command")

		invitePlatformManager = enablerplatform.GetInstance(&logger)
		// Loads the network configuration for the user.
		if err := invitePlatformManager.LoadUser("", userId); err != nil {
			return err
		}
		// Adds the passed organization to the network.
		// If multiple organizations, part of the network then it only endorses the transaction.

		// fmt.Printf("\n\n Adding the  '%s' for user '%s' has been Successfully created.\n", createPlatformManager.Enablers[0].NetworkName, userId)

		if err := invitePlatformManager.AddOrganization(useVolume, file, userLogging); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&userId, "userId", "u", "", "The User ID for the user.")
	addCmd.Flags().StringVarP(&file, "zipfile", "z", "", "zip file containing the relevant information.")
}
