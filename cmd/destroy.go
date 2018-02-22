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
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy instances",
	Long:  `Basically does a cleanup. Destroys all instances create by this tool`,
	Run: func(cmd *cobra.Command, args []string) {
		region, _ := cmd.Flags().GetString("region")
		enableVerbose, _ := cmd.Flags().GetBool("verbose")
		if enableVerbose {
			os.Setenv("__VERBOSE", "1")
		}

		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		verbose("Destroying all instances with tag:" + K8LInstanceTagName)
		instanceIDs := getAllInstanceIDsForTag(sess, K8LInstanceTagName)

		for i := range instanceIDs {
			instanceID := instanceIDs[i]
			destroyInstance(sess, instanceID)
		}

		fmt.Println("destroy called")
	},
}

func init() {
	inventoryCmd.AddCommand(destroyCmd)
}

func getAllInstanceIDsForTag(sess *session.Session, tag string) []string {
	svc := ec2.New(sess)

	verbose("Getting all instances for tag:" + tag)

	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:" + tag),
				Values: []*string{
					aws.String("yes"),
				},
			},
		},
	})

	if err != nil {
		handleError(err, "Error describing instances: ")
	}

	instanceIDs := make([]string, 0, 16)

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			instanceIDs = append(instanceIDs, aws.StringValue(instance.InstanceId))
		}
	}

	verbose("Found following instances:" + strings.Join(instanceIDs, ","))

	return instanceIDs
}

func destroyInstance(sess *session.Session, instanceID string) {
	svc := ec2.New(sess)

	verbose("Destroying instance:" + instanceID)

	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})

	if err != nil {
		handleError(err, "Error destorying instance: ")
	}

	verbose("Instance:" + instanceID + " destroyed")
}
