package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

type Tags []struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type DiscoveryConfig struct {
	Config struct {
		LoadBalancerName string    `yaml:"loadBalancerName"`
		Subnets          []*string `yaml:"subnets"`
		Region           string    `yaml:"region"`
		SecurityGroups   []*string `yaml:"securityGroups"`
		SslCert          string    `yaml:"sslCert"`
		Tags             []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"tags"`
		Route53 struct {
			Zone    string `yaml:"zone"`
			Enabled bool   `yaml:"enabled"`
		} `yaml:"route53"`
	} `yaml:"config"`
	Endpoints []struct {
		Name   string `yaml:"name"`
		Type   string `yaml:"type"`
		Search struct {
			Region string `yaml:"region"`
			Tags   []struct {
				Name  string `yaml:"name"`
				Value string `yaml:"value"`
			} `yaml:"tags"`
			Service struct {
				Name      string `yaml:"name"`
				Namespace string `yaml:"namespace"`
			} `yaml:"service"`
			DomainName string `yaml:"domainName"`
		} `yaml:"search"`
		DomainName string `yaml:"domainName"`
		Path       string `yaml:"path"`
		Port       int64  `yaml:"port"`
	} `yaml:"endpoints"`
}

var verbose bool
var local bool
var kubeconfig string
var discoveryConfig DiscoveryConfig
var configFile string

func init() {
	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.BoolVar(&verbose, "v", false, "Verbose")
	flag.BoolVar(&local, "l", false, "Run outside kube cluster (dev purposes)")

	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
}

func (c *DiscoveryConfig) getConf() *DiscoveryConfig {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

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

func main() {
	var lb *elbv2.LoadBalancer
	var listenerHttps *elbv2.Listener
	var listenerHttp *elbv2.Listener
	var listenerArn string

	flag.Parse()
	discoveryConfig.getConf()

	lbs := findALB(discoveryConfig.Config.LoadBalancerName, discoveryConfig.Config.Region)
	if len(lbs.LoadBalancers) < 1 {
		lb = createALB(
			discoveryConfig.Config.LoadBalancerName,
			discoveryConfig.Config.Subnets,
			discoveryConfig.Config.SecurityGroups,
			discoveryConfig.Config.Region)
		listeners := addDefaultListeners(*lb.LoadBalancerArn, "200", discoveryConfig.Config.SslCert, discoveryConfig.Config.Region)
		listenerHttps = listeners[0]
	} else {
		lb = lbs.LoadBalancers[0]
	}
	listenerHttp = findListener(*lb.LoadBalancerArn, 443, discoveryConfig.Config.Region)
	listenerHttps = findListener(*lb.LoadBalancerArn, 80, discoveryConfig.Config.Region)

	addAWSTags(
		discoveryConfig.Config.Tags,
		discoveryConfig.Config.LoadBalancerName,
		discoveryConfig.Config.Region)

	for i := range discoveryConfig.Endpoints {
		log.Printf("========== %s ============\n", discoveryConfig.Endpoints[i].Type)
		if discoveryConfig.Endpoints[i].Port == 443 {
			listenerArn = *listenerHttps.ListenerArn
		} else {
			listenerArn = *listenerHttp.ListenerArn
		}
		switch t := discoveryConfig.Endpoints[i].Type; t {
		case "es":
			{
				log.Println("Discovery of Elastic Search")
				es := discoverES(discoveryConfig.Endpoints[i].Search.DomainName, discoveryConfig.Endpoints[i].Search.Region)
				log.Printf("Setting up rules for %s\n", es)
				addRule(*lb.LoadBalancerArn,
					listenerArn,
					discoveryConfig.Endpoints[i].DomainName,
					es,
					discoveryConfig.Endpoints[i].Path,
					discoveryConfig.Endpoints[i].Search.Region)
			}
		case "ec2":
			{
				log.Println("Discovering EC2 instance using tags")
				ec2 := discoverEc2(
					discoveryConfig.Endpoints[i].Search.Tags,
					discoveryConfig.Endpoints[i].Search.Region)

				/* I can only do one redirect, pick up first one */
				if len(ec2) > 0 {
					addRule(*lb.LoadBalancerArn,
						listenerArn,
						discoveryConfig.Endpoints[i].DomainName,
						ec2[0],
						discoveryConfig.Endpoints[i].Path,
						discoveryConfig.Endpoints[i].Search.Region)
				}
			}
		case "k8s":
			log.Printf("Discovery of Kubernetes services: %s in %s namespace\n",
				discoveryConfig.Endpoints[i].Search.Service.Name,
				discoveryConfig.Endpoints[i].Search.Service.Namespace)
			srv, err := discoverK8sService(kubeconfig,
				discoveryConfig.Endpoints[i].Search.Service.Name,
				discoveryConfig.Endpoints[i].Search.Service.Namespace)
			if err != nil {
				log.Fatalln(err)
				continue
			}
			log.Println(srv.Status.LoadBalancer.Ingress[0].Hostname)
			addRule(*lb.LoadBalancerArn,
				listenerArn,
				discoveryConfig.Endpoints[i].DomainName,
				srv.Status.LoadBalancer.Ingress[0].Hostname,
				discoveryConfig.Endpoints[i].Path,
				discoveryConfig.Endpoints[i].Search.Region)

		default:
			{
				log.Printf("Unsupported type %s\n", t)
			}
		}
	}

}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
