package main

import (
	"flag"
	"fmt"
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
	versionFlag = flag.Bool("version", false, "print version")
	dryRun      = flag.Bool("dry-run", false, "dry run")
	minutes     = flag.Int("minutes", 0, "number of minutes to keep ec2 alive")
	hours       = flag.Int("hours", 0, "number of hours to keep ec2s alive")
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	log.Println("started deadline version", version)

	sess := must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	svc := ec2.New(sess)

	log.Printf("waiting for %d hours and %d minutes before terminating instances", *hours, *minutes)
	totalTime := (time.Hour * time.Duration(*hours)) + (time.Minute * time.Duration(*minutes))
	future := time.Now().Add(totalTime)

	for time.Now().Before(future) {
		var runningInstances int
		for _, res := range getReservations(svc) {
			runningInstances += len(res.Instances)
		}
		timeUntilTermination := time.Until(future).Truncate(time.Second).String()
		log.Printf("%s until termination, %d instances are running", timeUntilTermination, runningInstances)
		time.Sleep(time.Minute * 1)
	}

	for {
		var instancesToDelete []*string
		for _, res := range getReservations(svc) {
			for _, inst := range res.Instances {
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

func getReservations(svc *ec2.EC2) []*ec2.Reservation {
	return must(svc.DescribeInstances(&ec2.DescribeInstancesInput{MaxResults: aws.Int64(1000)})).Reservations
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
