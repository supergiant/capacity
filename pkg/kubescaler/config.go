package kubescaler

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/persistentfile"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
	"github.com/supergiant/capacity/pkg/log"
)

const (
	EnvPrefix = "CAPACITY"
)

type ConfigManager interface {
	SetConfig(api.Config) error
	PatchConfig(api.Config) error
	GetConfig() api.Config
	IsReady() bool
}

func Merge(c, patch api.Config) api.Config {
	if patch.Paused != nil {
		c.Paused = patch.Paused
	}
	// TODO: use pointers for it?
	if patch.WorkersCountMin != 0 {
		c.WorkersCountMin = patch.WorkersCountMin
	}
	if patch.WorkersCountMax != 0 {
		c.WorkersCountMax = patch.WorkersCountMax
	}
	if len(patch.MachineTypes) != 0 {
		c.MachineTypes = patch.MachineTypes
	}
	if len(patch.IgnoredNodeLabels) != 0 {
		c.IgnoredNodeLabels = patch.IgnoredNodeLabels
	}
	if patch.NewNodeTimeBuffer != 0 {
		c.NewNodeTimeBuffer = patch.NewNodeTimeBuffer
	}
	return c
}

type configManager struct {
	file persistentfile.Interface

	mu   sync.RWMutex
	conf api.Config
	isReady bool
}

func NewConfigManager(file persistentfile.Interface) (*configManager, error) {
	isReady := false
	conf := api.Config{}
	raw, err := file.Read()

	if err != nil {
		log.Warnf("Read config %v", err)
	} else {
		isReady = true
		// TODO: use codec to support more formats
		if err = json.Unmarshal(raw, &conf); err != nil {
			return nil, errors.Wrap(err, "decode config")
		}
	}

	return &configManager{
		file: file,
		mu:   sync.RWMutex{},
		isReady: isReady,
		conf: applyEnv(conf),
	}, nil
}

func (m *configManager) SetConfig(conf api.Config) error {
	if err := m.write(conf); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.isReady = true
	m.conf = conf
	return nil
}

func (m *configManager) PatchConfig(in api.Config) error {
	newConf := Merge(m.GetConfig(), in)
	if err := newConf.Validate(); err != nil {
		return err
	}
	return m.SetConfig(newConf)
}

func (m *configManager) write(conf api.Config) error {
	raw, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "encode config")
	}
	return errors.Wrap(m.file.Write(raw), "write config")
}

func (m *configManager) GetConfig() api.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conf
}


func (m *configManager) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isReady
}

// TODO: just a hack, use viper in the future
func applyEnv(conf api.Config) api.Config {
	envMap := map[string]string{
		aws.KeyID:     EnvPrefix + "_PROVIDER_AWS_KEYID",
		aws.SecretKey: EnvPrefix + "_PROVIDER_AWS_SECRETKEY",
	}

	for key, env := range envMap {
		val := os.Getenv(env)
		if val != "" {
			conf.Provider[key] = val
		}
	}
	return conf
}

// TODO: show this on cli help subcommand
func writeExampleConfig(file persistentfile.Interface) error {
	conf := &api.Config{
		SSHPubKey:         "REPLACE_IT",
		ClusterName:       "REPLACE_IT",
		MasterPrivateAddr: "REPLACE_IT",
		KubeAPIHost:       "REPLACE_IT",
		KubeAPIPort:       "REPLACE_IT",
		KubeAPIUser:       "REPLACE_IT",
		KubeAPIPassword:   "REPLACE_IT",
		ProviderName:      "aws",
		Provider: map[string]string{
			aws.KeyID:          "REPLACE_IT",
			aws.SecretKey:      "REPLACE_IT",
			aws.Region:         "REPLACE_IT",
			aws.KeyName:        "REPLACE_IT",
			aws.ImageID:        "ami-cc0900ac",
			aws.IAMRole:        "kubernetes-node",
			aws.SecurityGroups: strings.Join([]string{"REPLACE_IT"}, provider.ListSep),
			aws.SubnetID:       "REPLACE_IT",
			aws.VolType:        "gp2",
			aws.VolSize:        "100",
		},
		MachineTypes:    []string{"m4.large"},
		WorkersCountMax: 3,
		WorkersCountMin: 1,
		Paused:          BoolPtr(true),
	}

	raw, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	return file.Write(raw)
}

func BoolPtr(in bool) *bool {
	return &in
}
