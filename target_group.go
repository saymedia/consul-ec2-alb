package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/elbv2"
	consulApi "github.com/hashicorp/consul/api"
)

type TargetGroup struct {
	arn               string
	awsRegion         string
	albClient         *elbv2.ELBV2
	consulServiceName string
	consulClient      *consulApi.Client
}

func NewTargetGroup(
	config *TargetGroupConfig,
	consulConfig *ConsulConfig,
	awsConfig *AWSConfig,
) (*TargetGroup, error) {

	consulClient, err := consulApi.NewClient(
		consulConfig.AsAPIConfig(config.DatacenterName),
	)
	if err != nil {
		return nil, fmt.Errorf("consul config error: %s", err)
	}

	awsRegion := config.AWSRegion()
	if awsRegion == "" {
		return nil, fmt.Errorf("ARN malformed, so couldn't extract region name")
	}

	albClient, err := awsConfig.GetALBClient(awsRegion)
	if err != nil {
		return nil, err
	}

	return &TargetGroup{
		arn:               config.TargetGroupARN,
		awsRegion:         awsRegion,
		albClient:         albClient,
		consulServiceName: config.ServiceName,
		consulClient:      consulClient,
	}, nil
}

func (tg *TargetGroup) KeepSyncing() {
	fmt.Printf("This is where I would process %s\n", tg.arn)
}
