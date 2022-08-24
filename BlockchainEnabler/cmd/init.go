/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"BlockchainEnabler/BlockchainEnabler/internal/conf"
	"BlockchainEnabler/BlockchainEnabler/internal/enablerplatform"
	"BlockchainEnabler/BlockchainEnabler/internal/types"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var initOptions conf.InitializationOptions

var confFile bool
var selectedBlockchain string
var networkID string
var organizationName string

var promptNames bool
var platformManager *enablerplatform.EnablerPlatformManager
var basicSetup bool
var localSetup bool

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "This command is for initializing the network",
	Long: `The user has to provide the initialization parameters for the network to be initialized
	In Initialization Phase these things are taken care o9	1. Creating the yaml and the json files for running the setup.
	2. Verification and addition of the identities.
	3. Creation of the channel and the basic Block for the Blockchain. 
	4. The configuration file will be provided at the end of the setup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Here we need to check for the values provided by the user, storing these values and also validating these values.
		// first lets start with the number of members provided by the user along with the userid.
		// var userName string
		var userId string
		if err := checkBlockchainProvider(selectedBlockchain); err != nil {
			return err
		}
		logger.Printf("Initializing the network")
		if len(args) > 0 {
			userId = args[0]
			err := checkUserId(userId)
			if err != nil {
				return err
			}
		} else {
			userId, _ = prompt("User ID: ", checkUserId)
		}

		// var numOfMembers string
		// if len(args) > 1 {
		// 	numOfMembers = args[1]
		// 	if err := checkMembersCount(numOfMembers); err != nil {
		// 		return err
		// 	}
		// } else {
		// 	numOfMembers, _ = prompt("Number of members: ", checkMembersCount)
		// }
		memberCount := 1
		// memberCount, _ := strconv.Atoi(numOfMembers)
		initOptions.UserId = userId
		initOptions.OrgNames = make([]string, 0, memberCount)
		initOptions.NodeNames = make([]string, 0, memberCount)
		if networkID == "" {
			networkID = fmt.Sprintf("%s_network1", userId)
		}
		if organizationName == "" {
			organizationName = fmt.Sprintf("%sOrg1", userId)
		}
		initOptions.NetworkName = networkID
		initOptions.UseVolume = useVolume
		initOptions.OrgNames = append(initOptions.OrgNames, organizationName)
		initOptions.BasicSetup = basicSetup
		if promptNames {

		} else {
			for i := 0; i < memberCount; i++ {

				initOptions.NodeNames = append(initOptions.NodeNames, fmt.Sprintf("peer%d", i))
			}
		}
		initOptions.BlockchainType, _ = types.BlockchainProviderSelection(selectedBlockchain)
		initOptions.UserLogging = userLogging
		platformManager = enablerplatform.GetInstance(&logger)
		//  Initialization of the User Network.
		// Initilization of all the components needed to run, which involves the creation of the docker yaml file and other stuff.
		//  this will only create the docker yaml file wont be responsible for running the network.
		if err := platformManager.InitEnablerPlatform(userId, memberCount, &initOptions, localSetup); err != nil {
			return nil
		}
		// fmt.Printf("\n\nThe user '%s' has been Successfully initialized. To create the network, run:\n\n go run main.go create -u %s\n\n", userId, userId)
		return nil
	},
}

func checkBlockchainProvider(s string) error {
	blockchainSelected, err := types.BlockchainProviderSelection(s)
	if err != nil {
		return err
	}
	if blockchainSelected == types.Corda {
		return errors.New("Support for corda coming soon")
	}
	return nil
}

func checkUserId(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("Userid cannot be left blank")
	}
	return nil
}

func checkMembersCount(input string) error {
	if i, err := strconv.Atoi(input); err != nil {
		return errors.New("Number Invalid")
	} else if i <= 0 {
		return errors.New("Enter a positive number greater than 0")
	}

	return nil
}

func init() {

	rootCmd.AddCommand(initCmd)
	// initCmd.Flags().IntVarP(&meminfo.NumberOfMembers, "members", "m", 0, "Number of member organizations.")
	// initCmd.MarkFlagRequired("members")
	initCmd.Flags().StringVarP(&selectedBlockchain, "blockchain", "b", "fabric", fmt.Sprintf("Provide the Blockchain you would like to use options are %v", types.BlockchainProvidersList))
	initCmd.Flags().BoolVar(&promptNames, "prompt-names", false, "Prompt for org and node names")
	// initCmd.Flags().BoolVarP(&confFile, "conf", "f", false, "Configuration file")
	initCmd.Flags().StringVarP(&networkID, "networkID", "n", "", "Provide the name for the network.")
	initCmd.Flags().StringVarP(&organizationName, "orgName", "o", "", "Provide the name for the organization default value org1.")
	initCmd.Flags().BoolVarP(&basicSetup, "simpleSetup", "s", false, "Choose this to form a network without the orderer.")
	initCmd.Flags().BoolVar(&localSetup, "local", false, "Select this to run the setup on a local network.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
