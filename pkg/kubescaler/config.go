package capacity

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
)

type Config struct {
	SSHPubKey         string            `json:"sshPubKey"`
	ClusterName       string            `json:"clusterName"`
	MasterPrivateAddr string            `json:"masterPrivateAddr"`
	KubeAPIPort       string            `json:"kubeAPIPort"`
	KubeAPIPassword   string            `json:"kubeAPIPassword"`
	ProviderName      string            `json:"providerName"`
	Provider          map[string]string `json:"provider"`

	Stopped                 bool          `json:"stopped"`
	NodesCountMin           int           `json:"nodesCountMin"`
	NodesCountMax           int           `json:"nodesCountMax"`
	MachineTypes            []string      `json:"machineTypes"`
	MaxMachineProvisionTime time.Duration `json:"maxMachineProvisionTime"`
}

func (c *Config) Merge(in *Config) {
	switch {
	case in.NodesCountMin != 0:
		c.NodesCountMin = in.NodesCountMin
	case in.NodesCountMax != 0:
		c.NodesCountMax = in.NodesCountMax
	case in.MachineTypes != nil:
		c.MachineTypes = in.MachineTypes
	}
}

type PersistentConfig struct {
	filepath string

	mu   sync.RWMutex
	conf *Config
}

func NewPersistentConfig(filepath string) (*PersistentConfig, error) {
	rc, err := fileReadCloser(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			if err = writeExampleConfig(filepath); err != nil {
				return nil, err
			}
			if rc, err = fileReadCloser(filepath); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer rc.Close()

	conf := &Config{}
	if err = json.NewDecoder(rc).Decode(conf); err != nil {
		return nil, err
	}

	return &PersistentConfig{
		filepath: filepath,
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

	m.conf.Merge(in)
	return json.NewEncoder(wc).Encode(m.conf)
}

func (m *PersistentConfig) GetConfig() Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	return *m.conf
}

func fileReadCloser(filepath string) (io.ReadCloser, error) {
	return os.OpenFile(filepath, os.O_RDONLY, 0644)

}

func fileWriteCloser(filepath string) (io.WriteCloser, error) {
	return os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func writeExampleConfig(filepath string) error {
	conf := &Config{
		SSHPubKey:         "REPLACE_IT",
		ClusterName:       "REPLACE_IT",
		MasterPrivateAddr: "REPLACE_IT",
		KubeAPIPort:       "REPLACE_IT",
		KubeAPIPassword:   "REPLACE_IT",
		ProviderName:      "REPLACE_IT",
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
		MachineTypes: make([]string, 0),
	}

	fw, err := fileWriteCloser(filepath)
	if err != nil {
		return err
	}
	defer fw.Close()

	return json.NewEncoder(fw).Encode(conf)
}
