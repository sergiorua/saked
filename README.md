# Serg's AWS and Kubernetes Endpoint Discovery

## What is this for?

The problem I was trying to solve is that we destroy development and test infrastructures every night and bring it back in the morning.
This will cause all endpoints for any AWS services we use to be recreated with a different host name / URL. As we use some whitelisting
it's not in our best interest to have to update URL in our proxy whitelist every day.

This code will discover all the services you are creating and set up a single entry point, an AWS Application Load Balancer with 302 redirects to any
of these services.

## Is it worth the efford?

Perhaps not. There are multiple solutions for this problem. You could simply use DNS CNAMEs but some services such as AWS API Gateways don't work
with CNAMEs (though you can configure custom domains and this is not a problem then).

At the end of the day it has been a learning exercise for me on AWS SDK and Kube SDK using golang with some utility at the end.

## How do I use it?

Look at the [example](https://github.com/sergiorua/saked/blob/develop/example/discovery.yaml). 

Just compile it and run it.

Detailed instructions to come once is working properly.
