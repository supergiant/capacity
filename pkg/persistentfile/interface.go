package persistentfile

import (
	"github.com/supergiant/capacity/pkg/persistentfile/configmap"
	"github.com/supergiant/capacity/pkg/persistentfile/file"
)

var (
	_ Interface = &file.FSFile{}
	_ Interface = &configmap.CMFile{}
)

type Interface interface {
	Info() string
	Read() ([]byte, error)
	Write(data []byte) error
}
