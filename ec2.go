package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func discoverEc2(tags Tags, awsRegion string) []string {
	var filters []*ec2.Filter
	var ec2DnsNames []string

	for _, tag := range tags {
		f := ec2.Filter{
			Name: aws.String(fmt.Sprintf("tag:%v", tag.Name)),
			Values: []*string{
				aws.String(tag.Value),
			},
		}
		filters = append(filters, &f)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal(err)
		return ec2DnsNames
	}

	for _, res := range result.Reservations {
		for _, inst := range res.Instances {
			if len(*inst.PublicDnsName) > 1 {
				ec2DnsNames = append(ec2DnsNames, *inst.PublicDnsName)
			} else {
				ec2DnsNames = append(ec2DnsNames, *inst.PrivateDnsName)
			}
		}
	}

	return ec2DnsNames
}
