package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	consulApi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/hcl"
)

type Config struct {
	AWS          *AWSConfig           `hcl:"aws"`
	Consul       *ConsulConfig        `hcl:"consul"`
	TargetGroups []*TargetGroupConfig `hcl:"target_group"`
}

type TargetGroupConfig struct {
	TargetGroupARN string `hcl:",key"`
	ServiceName    string `hcl:"service"`
	DatacenterName string `hcl:"datacenter"`
}

type AWSConfig struct {
	AccessKeyID     string `hcl:"access_key_id"`
	SecretAccessKey string `hcl:"secret_access_key"`
	SecurityToken   string `hcl:"security_token"`
}

type ConsulConfig struct {
	Address string `hcl:"address"`
	Token   string `hcl:"token"`
	Scheme  string `hcl:"scheme"`
}

func LoadConfigFile(filename string) (*Config, error) {
	configSource, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %s", filename, err)
	}

	config := Config{}
	err = hcl.Unmarshal(configSource, &config)
	if err != nil {
		return nil, fmt.Errorf("syntax error in %s: %s", filename, err)
	}

	return &config, nil
}

// LoadConfigFiles loads several configuration files and merges them
// together into a single Config instance.
func LoadConfigFiles(filenames []string) (*Config, error) {
	config := &Config{}
	awsConfigSource := ""
	consulConfigSource := ""

	for _, filename := range filenames {
		oneConfig, err := LoadConfigFile(filename)
		if err != nil {
			return nil, err
		}

		config.TargetGroups = append(config.TargetGroups, oneConfig.TargetGroups...)

		if oneConfig.AWS != nil {
			if awsConfigSource != "" {
				return nil, fmt.Errorf(
					"'aws' block appears twice, in both %s and %s",
					awsConfigSource,
					filename,
				)
			}
			awsConfigSource = filename
			config.AWS = oneConfig.AWS
		}

		if oneConfig.Consul != nil {
			if consulConfigSource != "" {
				return nil, fmt.Errorf(
					"'consul' block appears twice, in both %s and %s",
					consulConfigSource,
					filename,
				)
			}
			consulConfigSource = filename
			config.Consul = oneConfig.Consul
		}
	}
	return config, nil
}

func (c *TargetGroupConfig) AWSRegion() string {
	arn := c.TargetGroupARN

	// The region is embedded in the ARN, assuming that we've been given
	// a valid and complete ARN. If not, we'll just return the empty string
	// and let the caller signal an error.
	parts := strings.Split(arn, ":")

	if len(parts) < 6 || parts[0] != "arn" {
		// Doesn't seem like a valid ARN
		return ""
	}

	return parts[3]
}

func (c *AWSConfig) GetCredentials() *awsCredentials.Credentials {
	providers := make([]awsCredentials.Provider, 0, 2)

	if c != nil {
		providers = append(providers, &awsCredentials.StaticProvider{
			Value: awsCredentials.Value{
				AccessKeyID:     c.AccessKeyID,
				SecretAccessKey: c.SecretAccessKey,
			},
		})
	}

	providers = append(providers, &awsCredentials.EnvProvider{})

	cfg := &aws.Config{}
	metadataClient := ec2metadata.New(awsSession.New(cfg))
	providers = append(providers, &ec2rolecreds.EC2RoleProvider{
		Client: metadataClient,
	})

	return awsCredentials.NewChainCredentials(providers)
}

func (c *AWSConfig) GetALBClient(region string) (*elbv2.ELBV2, error) {
	creds := c.GetCredentials()

	cfg := &aws.Config{
		Region:      aws.String(region),
		Credentials: creds,

		// We will do our own retrying
		MaxRetries: aws.Int(0),
	}

	sess, err := awsSession.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %s", err)
	}

	return elbv2.New(sess), nil
}

func (c *ConsulConfig) AsAPIConfig(datacenter string) *consulApi.Config {
	apiConfig := &consulApi.Config{
		Datacenter: datacenter,
	}

	if c != nil {
		apiConfig.Address = c.Address
		apiConfig.Scheme = c.Scheme
		apiConfig.Token = c.Token
	}

	return apiConfig
}
