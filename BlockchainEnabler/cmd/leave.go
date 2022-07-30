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

var leavePlatformEnabler *enablerplatform.EnablerPlatformManager

var orgID string
var networkName string
var userID string
var finalize bool

// leaveCmd represents the leave command
var leaveCmd = &cobra.Command{
	Use:   "leave",
	Short: "This command is for leaving the network ",
	Long: `This command is used to leave the network. 
	The user needs to provide the user_name and then the `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("leave called")
		leavePlatformEnabler = enablerplatform.GetInstance(&logger)
		leavePlatformEnabler.LoadUser("", userID)
		// logger.Printf(joinPlatformManager.UserId)

		leavePlatformEnabler.LeaveNetwork(networkName, orgID, useVolume, finalize)
	},
}

func init() {
	rootCmd.AddCommand(leaveCmd)
	leaveCmd.Flags().StringVarP(&userID, "userId", "u", "", "The User ID for the user.")

	leaveCmd.Flags().StringVarP(&orgID, "orgName", "o", "", "The organization name which wants to leave the channel.")
	leaveCmd.Flags().StringVarP(&networkName, "networkName", "n", "", "The Network the organization which wants to leave")
	leaveCmd.Flags().BoolVarP(&finalize, "finalize", "p", false, "The final step of leave need to be run by the other org.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// leaveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// leaveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
