SOURCES=saked.go alb.go k8s.go ec2.go tags.go es.go appsync.go
OUTPUT=saked
build:
	go build $(SOURCES)

run:
	go run $(SOURCES)
