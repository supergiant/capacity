package capacity

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
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

	Paused          *bool    `json:"paused,omitempty"`
	WorkersCountMin int      `json:"workersCountMin"`
	WorkersCountMax int      `json:"workersCountMax"`
	MachineTypes    []string `json:"machineTypes"`
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

	return nil
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
			return nil, errors.New("example config has generated on " + filepath + ". Please, go through REPLACE_IT fields")
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
		KubeAPIHost:       "REPLACE_IT",
		KubeAPIPort:       "REPLACE_IT",
		KubeAPIUser:       "REPLACE_IT",
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
