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
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("join called")
		// since we now have the name of the org we want to create, first step would be to generate the org file, and then create the definition file for the org.
		// Files currently needed
		// 1. crypto file to generate the cryptographic files for the org
		// 2. Generate the configtx.yaml file which defines the configurations
		// 3. Docker file to run the container for the peer.

		// Now when the user specifies the user id and the network, the network related configurations should be loaded,
		// After they are loaded, then we initialize the instance as a fabric / ethereum instance which could be then implemented.

		joinPlatformManager = enablerplatform.GetInstance(&logger)
		joinPlatformManager.LoadUser("", userId)
		// logger.Printf(invitePlatformManager.UserId)
		joinPlatformManager.JoinNetwork(useVolume, zipFile, basic)

	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
	// create two flag here 1 flag is for providing the name of the org which has to be joined and the next the name of the org1

	joinCmd.Flags().StringVarP(&userId, "userId", "u", "", "The User ID for the user.")
	joinCmd.Flags().StringVarP(&zipFile, "zipFile", "z", "", "The zip of the files needed.")
	joinCmd.Flags().BoolVarP(&basic, "simpleSetup", "s", false, "Function to enable or disable the use of Basic setup default: false")

	// joinCmd.Flags().StringVar(&configFile,"config","","Provide the config file needed to add Organization in network.")

	// joinCmd.Flags().BoolVarP(&finalize, "finalize", "f", false, "This phase runs on the behalf of the joining network.")

	// joinPlatformManager = enablerplatform.GetInstance(&logger)

	// It needs to check if the organization is already present or not if it is not present then create a default orgnatization with the name specified by the user.
	// once the join -c org3 -p org1 is done then
	// first it will check the org3 is present -> check throuh the user and the network and then bring up the network .
	// Currently would create the org3 files which are necessary and then generate the org3.json file in the folder

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// joinCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// joinCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
