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

var acceptPlatformManager *enablerplatform.EnablerPlatformManager

// acceptCmd represents the accept command
var acceptCmd = &cobra.Command{
	Use:   "accept",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("accept called")

		acceptPlatformManager = enablerplatform.GetInstance(&logger)
		acceptPlatformManager.LoadUser("", userId)
		// logger.Printf(invitePlatformManager.UserId)
		acceptPlatformManager.AcceptNetwork(networkId, orgName, useVolume)
	},
}

func init() {
	rootCmd.AddCommand(acceptCmd)
	acceptCmd.Flags().StringVarP(&userId, "userId", "u", "", "The User ID for the user.")

	acceptCmd.Flags().StringVarP(&orgName, "orgName", "o", "", "The organization name which wants to join the network.")
	acceptCmd.Flags().StringVarP(&networkId, "networkId", "n", "", "The Network the organization which wants to join another network.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// acceptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// acceptCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}