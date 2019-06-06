package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

/*
	Loops through the tags found for the volume and calls `setTag`
	to add it via the AWS api
*/
func addAWSTags(tags Tags, elbName string, awsRegion string) {
	alb := findALB(elbName, awsRegion)
	currentTags := getAlbTags(*alb.LoadBalancers[0].LoadBalancerArn, awsRegion)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}

	svc := elbv2.New(sess)

	for i := range tags {
		key := tags[i].Name
		value := tags[i].Value
		log.Printf("\tAdding tag: %s = %s\n", key, value)

		if !hasTag(currentTags.TagDescriptions[0].Tags, key, value) {
			setTag(svc, key, value, *alb.LoadBalancers[0].LoadBalancerArn)
		}
	}
}

/*
	AWS api call to set the tag
*/
func setTag(svc *elbv2.ELBV2, tagKey string, tagValue string, albArn string) bool {
	tags := &elbv2.AddTagsInput{
		ResourceArns: []*string{
			aws.String(albArn),
		},
		Tags: []*elbv2.Tag{
			{
				Key:   aws.String(tagKey),
				Value: aws.String(tagValue),
			},
		},
	}
	ret, err := svc.AddTags(tags)
	if err != nil {
		log.Fatal(err)
		return false
	}
	if verbose {
		log.Println(ret)
	}
	return true
}

/*
   Check if the tag is already set. It wouldn't be a problem if it is
   but if you're using cloudtrail it may be an issue seeing it
   being set all multiple times
*/
func hasTag(tags []*elbv2.Tag, Key string, value string) bool {
	for i := range tags {
		if *tags[i].Key == Key && *tags[i].Value == value {
			log.Printf("\t\tTag %s already set with value %s\n", *tags[i].Key, *tags[i].Value)
			return true
		}
	}
	return false
}
