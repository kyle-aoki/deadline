package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	version = "1.0.0"
)

var (
	dryRun  = flag.Bool("dry-run", false, "dry run")
	seconds = flag.Int("seconds", 0, "number of seconds to keep ec2 alive")
	minutes = flag.Int("minutes", 0, "number of minutes to keep ec2 alive")
	hours   = flag.Int("hours", 0, "number of hours to keep ec2s alive")
)

func main() {
	log.Println("started deadline version", version)
	flag.Parse()

	sess := must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	svc := ec2.New(sess)

	log.Printf("waiting for %d hours, %d minutes, and %d seconds before terminating instances", *hours, *minutes, *seconds)
	time.Sleep(time.Hour * time.Duration(*hours))
	time.Sleep(time.Minute * time.Duration(*minutes))
	time.Sleep(time.Second * time.Duration(*seconds))

	for {
		var instancesToDelete []*string
		dio := must(svc.DescribeInstances(&ec2.DescribeInstancesInput{MaxResults: aws.Int64(1000)}))
		for _, reservation := range dio.Reservations {
			for _, inst := range reservation.Instances {
				if *inst.State.Name == ec2.InstanceStateNameShuttingDown || *inst.State.Name == ec2.InstanceStateNameTerminated {
					continue
				}
				log.Println("found instance to terminate:", *inst.InstanceId)
				instancesToDelete = append(instancesToDelete, inst.InstanceId)
			}
		}
		if len(instancesToDelete) == 0 {
			log.Println("no remaining instances found, exiting")
			os.Exit(0)
		}
		if !*dryRun {
			_ = must(svc.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: instancesToDelete}))
		}
		time.Sleep(time.Second * 10)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func must[T any](t T, err error) T {
	check(err)
	return t
}
