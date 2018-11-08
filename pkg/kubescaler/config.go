package kubescaler

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/persistentfile"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
)

const (
	EnvPrefix = "CAPACITY"
)

type Config struct {
	ClusterName     string            `json:"clusterName"`
	ProviderName    string            `json:"providerName"`
	Provider        map[string]string `json:"provider"`
	Paused          *bool             `json:"paused,omitempty"`
	PauseLock       bool              `json:"pauseLock"`
	ScanInterval    string            `json:"scanInterval"`
	WorkersCountMin int               `json:"workersCountMin"`
	WorkersCountMax int               `json:"workersCountMax"`
	MachineTypes    []string          `json:"machineTypes"`
	// TODO: this is hardcoded and doesn't used at the moment
	MaxMachineProvisionTime string            `json:"maxMachineProvisionTime"`
	IgnoredNodeLabels       map[string]string `json:"ignoredNodeLabels"`
	NewNodeTimeBuffer       int               `json:"newNodeTimeBuffer"`

	// These is a SG1.0 UserData template parameters
	// TODO: add an explicit struct for it or use a map for dynamic values
	MasterPrivateAddr string `json:"masterPrivateAddr"`
	KubeAPIHost       string `json:"kubeAPIHost"`
	KubeAPIPort       string `json:"kubeAPIPort"`
	KubeAPIUser       string `json:"kubeAPIUser"`
	KubeAPIPassword   string `json:"kubeAPIPassword"`
	SSHPubKey         string `json:"sshPubKey"`
}

func (c Config) Validate() error {
	// TODO: pass it with a pointer or use the ConfigRequest struct for patches.
	if c.WorkersCountMin < 0 {
		return errors.New("WorkersCountMin can't be negative")
	}
	if c.WorkersCountMax < 0 {
		return errors.New("WorkersCountMax can't be negative")
	}
	return nil
}

func Merge(c, patch Config) Config {
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

type ConfigManager struct {
	file persistentfile.Interface

	mu   sync.RWMutex
	conf Config
}

func NewConfigManager(file persistentfile.Interface) (*ConfigManager, error) {
	raw, err := file.Read()
	if err != nil {
		if persistentfile.IsNotExist(err) {
			return nil, errors.Wrapf(err, "read config from %s", file.Info())
		}
		return nil, errors.Wrap(err, "get config")
	}

	conf := Config{}
	// TODO: use codec to support more formats
	if err = json.Unmarshal(raw, &conf); err != nil {
		return nil, errors.Wrap(err, "decode config")
	}

	return &ConfigManager{
		file: file,
		mu:   sync.RWMutex{},
		conf: applyEnv(conf),
	}, nil
}

func (m *ConfigManager) SetConfig(conf Config) error {
	if err := m.write(conf); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.conf = conf
	return nil
}

func (m *ConfigManager) PatchConfig(in Config) error {
	newConf := Merge(m.GetConfig(), in)
	if err := newConf.Validate(); err != nil {
		return err
	}
	return m.SetConfig(newConf)
}

func (m *ConfigManager) write(conf Config) error {
	raw, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "encode config")
	}
	return errors.Wrap(m.file.Write(raw), "write config")
}

func (m *ConfigManager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conf
}

// TODO: just a hack, use viper in the future
func applyEnv(conf Config) Config {
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
	conf := &Config{
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
