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
	"fmt"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "This command is for setting up and running the network",
	Long: `Some steps involve while running this network are:
	1. Checking if the docker is present in the host machine or not.
	2. Loading the enabler information -> regarding the members.
	3. Creating the gensis block and the configuration files.
	4. Running the containers for the whole -> orderer, ca, peer and other if needed.`,
	
RunE: func(cmd *cobra.Command, args []string) error{
		fmt.Println("create called")
		if err := docker.CheckDockerConfig(); err != nil {
			return err
		}

		// steps to follow the user specifies the name of the platform and then we run its containers.
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
