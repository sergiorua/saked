package main

import (
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

func getAlbTags(lbArn string, awsRegion string) *elbv2.DescribeTagsOutput {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)
	input := &elbv2.DescribeTagsInput{
		ResourceArns: []*string{
			aws.String(lbArn),
		},
	}
	result, err := svc.DescribeTags(input)
	if err != nil {
		log.Fatal(err)
		var e *elbv2.DescribeTagsOutput
		return e
	}

	return result
}

func findALB(lbName string, awsRegion string) *elbv2.DescribeLoadBalancersOutput {
	/* Connect to AWS */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)
	input := &elbv2.DescribeLoadBalancersInput{
		Names: []*string{
			aws.String(lbName),
		},
	}

	result, err := svc.DescribeLoadBalancers(input)
	if err != nil {
		log.Printf("Load balancer %s not found\n", lbName)
		return result
	}

	return result
}

func createALB(lbName string, subnets []*string, securityGroups []*string, awsRegion string) *elbv2.LoadBalancer {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)
	input := &elbv2.CreateLoadBalancerInput{
		Name:           aws.String(lbName),
		Subnets:        subnets,
		Scheme:         aws.String("internal"),
		SecurityGroups: securityGroups,
	}
	result, err := svc.CreateLoadBalancer(input)
	if err != nil {
		log.Fatalln(err)
	}

	return result.LoadBalancers[0]
}

func listernetExists(lbArn string, endpoint string, awsRegion string) bool {
	/* Connect to AWS */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)
	input := &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbArn),
	}
	result, err := svc.DescribeListeners(input)
	if err != nil {
		log.Println("Load balancer not found")
		log.Println(err)
		return false
	}

	for i := range result.Listeners {
		l := result.Listeners[i]
		for j := range l.DefaultActions {
			a := l.DefaultActions[j]

			if *a.RedirectConfig.Host == endpoint {
				log.Printf("%s already exists", *a.RedirectConfig.Host)
				return true
			}

		}
	}
	return false
}

// FIXME: filter by port!
func findListener(lbArn string, port int64, awsRegion string) *elbv2.Listener {
	var listener *elbv2.Listener

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}

	svc := elbv2.New(sess)
	input := &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbArn),
	}
	result, err := svc.DescribeListeners(input)
	if err != nil {
		log.Fatal(err)
		return listener
	}
	for i := range result.Listeners {
		listener = result.Listeners[i]
		if *listener.Port == port {
			return listener
		}
	}

	return listener
}

func addDefaultListeners(lbArn string, returnCode string, sslCert string, awsRegion string) []*elbv2.Listener {
	/* Connect to AWS */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)

	input := &elbv2.CreateListenerInput{
		Certificates: []*elbv2.Certificate{
			{
				CertificateArn: aws.String(sslCert),
			},
		},
		DefaultActions: []*elbv2.Action{
			{
				FixedResponseConfig: &elbv2.FixedResponseActionConfig{
					ContentType: aws.String("application/json"),
					MessageBody: aws.String(`{"message": "No endpoint found"}`),
					StatusCode:  aws.String(returnCode),
				},
				Type: aws.String("fixed-response"),
			},
		},
		Port:            aws.Int64(443),
		Protocol:        aws.String("HTTPS"),
		SslPolicy:       aws.String("ELBSecurityPolicy-2015-05"),
		LoadBalancerArn: aws.String(lbArn),
	}
	result, err := svc.CreateListener(input)
	if err != nil {
		log.Fatal(err)
	}

	/* repeat for HTTP */
	input = &elbv2.CreateListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				FixedResponseConfig: &elbv2.FixedResponseActionConfig{
					ContentType: aws.String("application/json"),
					MessageBody: aws.String(`{"message": "No endpoint found"}`),
					StatusCode:  aws.String(returnCode),
				},
				Type: aws.String("fixed-response"),
			},
		},
		Port:            aws.Int64(80),
		Protocol:        aws.String("HTTP"),
		LoadBalancerArn: aws.String(lbArn),
	}
	result, err = svc.CreateListener(input)
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		log.Println(result)
	}
	return result.Listeners
}

func ruleExists(listenerArn string, targetHostHeader string, endpoint string, path string, awsRegion string) (bool, int64) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)
	input := &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(listenerArn),
	}
	result, err := svc.DescribeRules(input)
	if err != nil {
		return false, 0
	}
	var priority int64
	var found bool

	found = false

	for _, rule := range result.Rules {
		if *rule.Priority == "default" {
			continue
		}
		x, err := strconv.ParseInt(*rule.Priority, 10, 64)
		if err != nil {
			log.Fatal(err)
			continue
		}
		if x > priority {
			priority = x
		}
		for _, cond := range rule.Conditions {
			log.Println("Condition Value: ")
			for _, v := range cond.HostHeaderConfig.Values {
				if *v == targetHostHeader {
					log.Printf("\t%s == %s\n", *v, targetHostHeader)
					found = true
				}
			}
		}
	}
	return found, priority
}

func addRule(lbArn string, listenerArn string, targetHostHeader string, endpoint string, path string, awsRegion string) {

	found, priority := ruleExists(listenerArn, targetHostHeader, endpoint, path, awsRegion)
	if found {
		return
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		panic(err)
	}
	svc := elbv2.New(sess)

	input := &elbv2.CreateRuleInput{
		Actions: []*elbv2.Action{
			{
				RedirectConfig: &elbv2.RedirectActionConfig{
					Host:       aws.String(endpoint),
					StatusCode: aws.String("HTTP_302"),
					Protocol:   aws.String("HTTPS"),
					Port:       aws.String("443"),
					Path:       aws.String("/"),
				},
				Type: aws.String("redirect"),
			},
		},
		Conditions: []*elbv2.RuleCondition{
			{
				Field: aws.String("host-header"),
				Values: []*string{
					aws.String(targetHostHeader),
				},
			},
		},
		ListenerArn: aws.String(listenerArn),
		Priority:    aws.Int64(priority + 1),
	}

	result, err := svc.CreateRule(input)
	if err != nil {
		log.Println("Rule not created")
		log.Println(err)
	}

	if verbose {
		log.Println(result)
	}
}
