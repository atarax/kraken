// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "Sends http-requests",
	Long: `Basically just sends http-requests with some little features like 
parallel execution and delay between requests.`,

	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")
		parallelism, _ := cmd.Flags().GetInt("parallelism")
		target, _ := cmd.Flags().GetString("target")

		if target == "" {
			fmt.Println("Need argument: target (URL)")
			os.Exit(1)
		}

		httpStress(count, parallelism, target)
	},
}

func httpStress(count int, parallelism int, target string) {
	barrier := make(chan bool)

	for i := 0; i < parallelism; i++ {
		go func() {
			for j := 0; j < count; j++ {
				http.Get(target)
			}
			barrier <- true
		}()
	}

	for i := 0; i < parallelism; i++ {
		_ = <-barrier
	}
}

func init() {
	stressCmd.Flags().IntP("parallelism", "p", 1, "Number of parallel executions")
	stressCmd.Flags().IntP("count", "c", 1, "Number of requests to be sent")
	stressCmd.Flags().IntP("delay", "d", 0, "Delay between requests")
	stressCmd.Flags().StringP("target", "t", "", "Target-URL")

	rootCmd.AddCommand(stressCmd)
}
