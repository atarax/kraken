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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)

/** K8LSecurityGroupName string name of the k8l-security-group */
const K8LSecurityGroupName = "k8l-sg"
const K8LInstanceTagName = "K8L_GUNPOWDER"
const K8LInstanceTagValue = "yes"

// acquireCmd represents the acquire command
var acquireCmd = &cobra.Command{
	Use:   "acquire",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("acquire called")

		amiPerRegion := map[string]string{
			"eu-central-1": "ami-5055cd3f",
			"eu-west-1":    "ami-1b791862",
			"eu-west-3":    "ami-c1cf79bc",
		}

		// var command, region, instanceID string
		var enableVerbose bool

		region, _ := cmd.Flags().GetString("region")
		// if err != nil {
		// 	handleError(err, "Error handling flag: region")
		// }
		// region := "eu-west-1"
		if enableVerbose {
			os.Setenv("__VERBOSE", "1")
		}

		if region == "" {
			fmt.Println("No region specified.")
			os.Exit(1)
		}

		verbose("Configured AWS-Region:" + region)

		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		securityGroupID := ensureSecurityGroup(sess, K8LSecurityGroupName)
		publicIP := createInstance(sess, amiPerRegion[region], securityGroupID, K8LSecurityGroupName)

		fmt.Println(publicIP)
	},
}

func init() {
	inventoryCmd.AddCommand(acquireCmd)

	acquireCmd.Flags().StringP("region", "r", os.Getenv("AWS_REGION"), "AWS-Region")
}

func ensureSecurityGroup(sess *session.Session, groupName string) string {

	groupFound := true

	groups, err := getSecurityGroups(sess, []string{groupName})

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "InvalidGroup.NotFound":
			groupFound = false
			verbose("Security-Group not found...creating one")
		default:
			handleError(err, "Error getting security-groups:")
		}
	}

	if !groupFound {
		group := createSecurityGroup(sess, groupName)
		fmt.Println(group)
		verbose("Security-Group created")

		time.Sleep(time.Duration(20) * time.Second)

		attachSecurityGroupRules(sess, groupName)
		verbose("Security-Group-Rules attached")

		return aws.StringValue(group.GroupId)
	}

	return aws.StringValue(groups.SecurityGroups[0].GroupId)
}

func getSecurityGroups(sess *session.Session, groupNames []string) (*ec2.DescribeSecurityGroupsOutput, error) {
	svc := ec2.New(sess)

	result, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupNames: aws.StringSlice(groupNames),
	})

	return result, err
}

func createSecurityGroup(sess *session.Session, name string) *ec2.CreateSecurityGroupOutput {
	input := &ec2.CreateSecurityGroupInput{
		Description: &name,
		GroupName:   &name,
	}
	ec2 := ec2.New(sess)

	out, err := ec2.CreateSecurityGroup(input)
	if err != nil {
		handleError(err, "Error creating security-groups:")
	}

	return out
}

func attachSecurityGroupRules(sess *session.Session, groupName string) {
	svc := ec2.New(sess)

	_, err := svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupName: aws.String(groupName),
		IpPermissions: []*ec2.IpPermission{
			(&ec2.IpPermission{}).
				SetIpProtocol("tcp").
				SetFromPort(0).
				SetToPort(65535).
				SetIpRanges([]*ec2.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				}),
		},
	})

	if err != nil {
		handleError(err, "Error attaching security-group-roles:")
	}
}

func createInstance(sess *session.Session,
	ami string,
	securityGroupID string,
	securityGroupName string,
) string {

	svc := ec2.New(sess)

	verbose("Creating instance, using ami:" + ami + ", security-group:" + securityGroupID)

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String(ami),
		InstanceType:     aws.String("t2.micro"),
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		SecurityGroupIds: aws.StringSlice([]string{securityGroupID}),
		SecurityGroups:   aws.StringSlice([]string{securityGroupName}),
		KeyName:          aws.String("home"),
	})

	if err != nil {
		handleError(err, "Error creating instance:")
	}

	instanceID := *runResult.Instances[0].InstanceId
	verbose("Created instance:" + instanceID)

	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(K8LInstanceTagName),
				Value: aws.String(K8LInstanceTagValue),
			},
		},
	})
	if err != nil {
		handleError(err, "Error tagging instance:")
	}

	verbose("Successfully tagged instance")
	verbose("Waiting for instance to be ready...")

	err = svc.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {
		handleError(err, "Error while waiting until instance is ready:")
	}

	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})

	if err != nil {
		handleError(err, "Error describing instances:")
	}

	publicIP := aws.StringValue(result.Reservations[0].Instances[0].PublicIpAddress)
	verbose("Instance:" + instanceID + " with Public-IP:" + publicIP + "is ready")

	return publicIP
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func verbose(message string) {
	if os.Getenv("__VERBOSE") == "1" {
		t := time.Now()
		fmt.Println(t.Format("2006-01-02 15:04:05"), " - ", message)
	}
}
