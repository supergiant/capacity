package aws

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/supergiant/capacity/pkg/providers"
	"github.com/saheienko/supergiant/pkg/clouds/aws"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Provider name:
const (
	Name = "aws"
) // AWS instance parameters:
const (
	KeyID          = "awsKeyID"
	SecretKey      = "awsSecretKey"
	Region         = "awsRegion"
	InstanceID     = "awsInstanceID"
	InstanceName   = "awsInstanceName"
	InstanceType   = "awsInstanceType"
	KeyName        = "awsKeyName"
	ImageID        = "awsImageID"
	IAMRole        = "awsIAMRole"
	SecurityGroups = "awsSecurityGroups"
	SubnetID       = "awsSubnetID"
	VolType        = "awsVolType"
	VolSize        = "awsVolSize"
	Tags           = "awsTags"
)

type Config struct {
	KeyName        string
	ImageID        string
	IAMRole        string
	SecurityGroups []*string
	SubnetID       string
	VolType        string
	VolSize        int64
	Tags           map[string]string
}

type AWSProvider struct {
	region   string
	instConf Config
	client   *aws.Client
}

func New(config providers.Config) (*AWSProvider, error) {
	// TODO: parse and validate config
	key, secret, region := config[KeyID], config[SecretKey], config[Region]

	client, err := aws.New(key, secret, providers.ParseMap(config[Tags]))
	if err != nil {
		return nil, err
	}

	return &AWSProvider{
		region: region,
		instConf: Config{
			KeyName:        config[KeyName],
			ImageID:        config[ImageID],
			IAMRole:        config[IAMRole],
			SecurityGroups: providers.ParseList(config[SecurityGroups]),
			SubnetID:       config[SubnetID],
			VolType:        config[VolType],
			VolSize:        int64(100),
			Tags:           providers.ParseMap(config[Tags]),
		},
		client: client,
	}, nil
}

func (p *AWSProvider) Name() string {
	return "aws"
}

func (p *AWSProvider) Machines(ctx context.Context) ([]*providers.Machine, error) {
	insts, err := p.client.ListRegionInstances(ctx, p.region, nil)
	if err != nil {
		return nil, nil
	}

	machines := make([]*providers.Machine, len(insts))
	for i := range insts {
		machines[i] = &providers.Machine{
			ID:        *insts[i].InstanceId,
			Name:      getName(insts[i].Tags),
			Type:      *insts[i].InstanceType,
			CreatedAt: *insts[i].LaunchTime,
		}
	}

	return machines, nil
}

func (p *AWSProvider) AvailableMachineTypes(ctx context.Context) ([]*providers.MachineType, error) {
	instTypes, err := p.client.AvailableInstanceTypes(ctx)
	if err != nil {
		return nil, err
	}

	mTypes := make([]*providers.MachineType, len(instTypes))
	for i := range instTypes {
		mem, err := parseMemory(instTypes[i].Attributes.Memory)
		if err != nil {
			return nil, err
		}
		cpu, err := parseVCPU(instTypes[i].Attributes.Memory)
		if err != nil {
			return nil, err
		}
		mTypes[i] = &providers.MachineType{
			Name:   instTypes[i].Attributes.InstanceType,
			Memory: mem,
			CPU:    cpu,
		}
	}

	return mTypes, nil
}

func (p *AWSProvider) CreateMachine(ctx context.Context, clusterName, name, mtype, userData string, config providers.Config) error {
	// TODO: merge and validate config parameters

	return p.client.CreateInstance(ctx, aws.InstanceConfig{
		ClusterName:    clusterName,
		Name:           name,
		Type:           mtype,
		Region:         p.region,
		ImageID:        p.instConf.ImageID,
		KeyName:        p.instConf.KeyName,
		IAMRole:        p.instConf.IAMRole,
		SecurityGroups: p.instConf.SecurityGroups,
		SubnetID:       p.instConf.SubnetID,
		VolumeType:     p.instConf.VolType,
		VolumeSize:     p.instConf.VolSize,
		Tags:           p.instConf.Tags,
		UsedData:       base64.StdEncoding.EncodeToString([]byte(userData)),
	})
}

func (p *AWSProvider) DeleteMachine(ctx context.Context, id string) error {
	return p.client.DeleteInstance(ctx, p.region, id)
}

func normalizeMemory(memory string) string {
	// "1 GiB" --> "1Gi"
	return strings.Trim(memory, " B")
}

func parseMemory(memory string) (resource.Quantity, error) {
	return resource.ParseQuantity(normalizeMemory(memory))
}

func parseVCPU(vcpu string) (resource.Quantity, error) {
	return resource.ParseQuantity(vcpu)
}

func getName(tags []*ec2.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

func parseVolSize(size string) (int64, error) {
	return strconv.ParseInt(size, 10, 64)
}
