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

var invitePlatformManager *enablerplatform.EnablerPlatformManager
var file string
var ordererInfo string

// inviteCmd represents the invite command
var inviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("invite called")

		invitePlatformManager = enablerplatform.GetInstance(&logger)
		invitePlatformManager.LoadUser("", userId)
		// logger.Printf(invitePlatformManager.UserId)
		invitePlatformManager.InviteOrganization(useVolume, file)
	},
}

// ALSO INSIDE THE INVITE command we need something which would create the sign and update part.

func init() {
	rootCmd.AddCommand(inviteCmd)
	inviteCmd.Flags().StringVarP(&userId, "userId", "u", "", "The User ID for the user.")
	inviteCmd.Flags().StringVarP(&file, "zipfile", "z", "", "zip file containing the relevant information.")
	// inviteCmd.Flags().StringVarP(&ordererInfo, "orderercaFile", "c", "", "Pass the orderer ca file only needed in fabric")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// inviteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// inviteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
