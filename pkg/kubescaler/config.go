package capacity

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
)

const (
	EnvPrevix = "CAPACITY"
)

type Config struct {
	SSHPubKey               string            `json:"sshPubKey"`
	ClusterName             string            `json:"clusterName"`
	MasterPrivateAddr       string            `json:"masterPrivateAddr"`
	KubeAPIHost             string            `json:"kubeAPIHost"`
	KubeAPIPort             string            `json:"kubeAPIPort"`
	KubeAPIUser             string            `json:"kubeAPIUser"`
	KubeAPIPassword         string            `json:"kubeAPIPassword"`
	ProviderName            string            `json:"providerName"`
	Provider                map[string]string `json:"provider"`
	ScanInterval            string            `json:"scanInterval"`
	MaxMachineProvisionTime string            `json:"maxMachineProvisionTime"`

	Paused            *bool             `json:"paused,omitempty"`
	PauseLock         bool              `json:"pauseLock"`
	WorkersCountMin   int               `json:"workersCountMin"`
	WorkersCountMax   int               `json:"workersCountMax"`
	MachineTypes      []string          `json:"machineTypes"`
	IgnoredNodeLabels map[string]string `json:"ignoredNodeLabels"`
}

func (c *Config) Merge(in *Config) error {
	if in.Paused != nil {
		c.Paused = in.Paused
	}
	if in.WorkersCountMin != 0 {
		if in.WorkersCountMin < 0 {
			return errors.New("WorkersCountMin can't be negative")
		}
		c.WorkersCountMin = in.WorkersCountMin
	}
	if in.WorkersCountMax != 0 {
		if in.WorkersCountMax < 0 {
			return errors.New("WorkersCountMax can't be negative")
		}
		c.WorkersCountMax = in.WorkersCountMax
	}
	if len(in.MachineTypes) != 0 {
		c.MachineTypes = in.MachineTypes
	}
	if len(in.IgnoredNodeLabels) != 0 {
		c.IgnoredNodeLabels = in.IgnoredNodeLabels
	}

	return nil
}

type PersistentConfig struct {
	filepath string

	mu   sync.RWMutex
	conf *Config
}

func NewPersistentConfig(fullpath string) (*PersistentConfig, error) {
	_, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		if err = writeExampleConfig(fullpath); err != nil {
			return nil, errors.Wrap(err, "write the example config")
		}
		return nil, errors.New("example config has generated on " + fullpath + ". Please, go through REPLACE_IT fields")
	}

	conf, err := readFileEnv(fullpath)
	if err != nil {
		return nil, errors.Wrap(err, "from file/env")
	}

	// update a persistent config
	wc, err := fileWriteCloser(fullpath)
	if err != nil {
		return nil, err
	}
	defer wc.Close()
	if err = json.NewEncoder(wc).Encode(conf); err != nil {
		return nil, err
	}

	return &PersistentConfig{
		filepath: fullpath,
		mu:       sync.RWMutex{},
		conf:     conf,
	}, nil
}

func (m *PersistentConfig) SetConfig(conf Config) error {
	wc, err := fileWriteCloser(m.filepath)
	if err != nil {
		return err
	}
	defer wc.Close()

	if err = json.NewEncoder(wc).Encode(conf); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.conf = &conf
	return nil
}

func (m *PersistentConfig) PatchConfig(in *Config) error {
	wc, err := fileWriteCloser(m.filepath)
	if err != nil {
		return err
	}
	defer wc.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	if err = m.conf.Merge(in); err != nil {
		return err
	}

	return json.NewEncoder(wc).Encode(m.conf)
}

func (m *PersistentConfig) GetConfig() Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	return *m.conf
}

func readFileEnv(fullpath string) (*Config, error) {
	data, err := ioutil.ReadFile(fullpath)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	if err = json.Unmarshal(data, conf); err != nil {
		return nil, errors.Wrap(err, "decode")
	}
	if conf.Provider == nil {
		conf.Provider = make(map[string]string)
	}
	if err := applyEnv(conf); err != nil {
		return nil, errors.Wrap(err, "applyEnv")
	}

	return conf, nil
}

// TODO: just a hack, use viper in the future
func applyEnv(conf *Config) error {
	envMap := map[string]string{
		aws.KeyID:     EnvPrevix + "_PROVIDER_AWS_KEYID",
		aws.SecretKey: EnvPrevix + "_PROVIDER_AWS_SECRETKEY",
	}

	for key, env := range envMap {
		val := os.Getenv(env)
		if val != "" {
			conf.Provider[key] = val
		}
	}

	return nil
}

func fileWriteCloser(filepath string) (io.WriteCloser, error) {
	return os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func writeExampleConfig(filepath string) error {
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

	fw, err := fileWriteCloser(filepath)
	if err != nil {
		return err
	}
	defer fw.Close()

	return json.NewEncoder(fw).Encode(conf)
}

func BoolPtr(in bool) *bool {
	return &in
}
