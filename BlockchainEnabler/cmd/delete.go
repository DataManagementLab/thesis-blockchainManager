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

var deletePlatformManager *enablerplatform.EnablerPlatformManager

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete command removes the resources used by the network and deletes the network and the organization.",
	Long: `Delete command performs the following steps:
	1. Stops the running containers.
	2. Removes the resources acquired during this phase.
	3. Clear the folders which were used by the organization.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running delete command")

		deletePlatformManager = enablerplatform.GetInstance(&logger)
		// steps to follow the user specifies the name of the platform and then we run its containers.
		// it needs to load in the basic file from the directory and initialize it with the values for the network.
		// We need to then check which kind of network it is and then we would call the network functions(objects).
		deletePlatformManager.LoadUser(networkId, userId)
		deletePlatformManager.DeleteNetwork(userLogging)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&userId, "userId", "u", "", "Provide the user Id for the network you want to run.")
	deleteCmd.Flags().StringVarP(&networkId, "netid", "n", "", "Provide the network id of the network you want to run.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
