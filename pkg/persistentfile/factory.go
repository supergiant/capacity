package persistentfile

import (
	"os"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/supergiant/capacity/pkg/persistentfile/configmap"
	"github.com/supergiant/capacity/pkg/persistentfile/file"
)

type FileProvider string

const (
	FSFile        FileProvider = "fs"
	ConfigMapFile FileProvider = "cm"
)

func IsNotExist(err error) bool {
	return file.IsNotExist(err) || configmap.IsNotExist(err)
}

type Config struct {
	Type FileProvider
	// File parameters
	Path string
	Perm os.FileMode
	// ConfigMap parameters
	ConfigMapName      string
	ConfigMapNamespace string
	Key                string
	ConfigMapClient    v1.ConfigMapsGetter
}

func New(c Config) (Interface, error) {
	switch c.Type {
	case FSFile:
		return file.New(c.Path, c.Perm)
	case ConfigMapFile:
		return configmap.New(c.ConfigMapName, c.ConfigMapNamespace, c.Key, c.ConfigMapClient)
	}
	return nil, errors.New("unknown file provider")
}
