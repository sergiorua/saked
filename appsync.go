package main

import (
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appsync"
)

func discoverAppsync(apiName string, awsRegion string) string {
	/* Connect to AWS */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}

	svc := appsync.New(sess)
	params := &appsync.ListGraphqlApisInput{}
	resp, err := svc.ListGraphqlApis(params)
	if err != nil {
		log.Fatalln(err)
	}
	for _, api := range resp.GraphqlApis {
		if *api.Name == apiName {
			for _, uri := range api.Uris {
				u, err := url.Parse(*uri)
				if err != nil {
					log.Fatalf("Cannot parse URL %s\n", u)
					log.Println(err)
				}
				return u.Hostname()
			}
		}
	}

	return ""
}
