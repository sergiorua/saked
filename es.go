package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticsearchservice"
)

func discoverES(domainName string, awsRegion string) string {
	/* Connect to AWS */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}

	svc := elasticsearchservice.New(sess)
	params := &elasticsearchservice.DescribeElasticsearchDomainInput{
		DomainName: aws.String(domainName),
	}
	resp, err := svc.DescribeElasticsearchDomain(params)
	if err != nil {
		log.Fatalln(err)
	}
	return *resp.DomainStatus.Endpoints["vpc"]
}
