package configmap

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/core/v1"
)

// Package specific errors:
var (
	ErrKeyNotFound            = errors.New("key not found")
	ErrInvalidConfigMap       = errors.New("configMap name and namespase should be provided")
	ErrInvalidConfigMapKey    = errors.New("data key for configMap should be provided")
	ErrInvalidConfigMapClient = errors.New("configMap client should be provided")
)

// IsNotExist returns a boolean indicating whether the error is known to report
// that kubernetes resource or known key isn't found.
func IsNotExist(err error) bool {
	return apierrors.IsNotFound(err) || errors.Cause(err) == ErrKeyNotFound
}

// CMFile represents a file on kubernetes ConfigMap.
type CMFile struct {
	cmName      string
	cmNamespace string
	key         string
	cmCreate    bool
	client      v1.ConfigMapsGetter
}

// New creates the new CMFile.
func New(cmName, ns, key string, client v1.ConfigMapsGetter) (*CMFile, error) {
	cmName, ns, key = strings.TrimSpace(cmName), strings.TrimSpace(ns), strings.TrimSpace(key)
	if cmName == "" || ns == "" {
		return nil, ErrInvalidConfigMap
	}
	if key == "" {
		return nil, ErrInvalidConfigMapKey
	}
	if client == nil {
		return nil, ErrInvalidConfigMapClient
	}

	return &CMFile{
		cmName:      cmName,
		cmNamespace: ns,
		key:         key,
		client:      client,
	}, nil
}

// Info describes a file stored on kubernetes ConfigMap.
func (f CMFile) Info() string {
	return fmt.Sprintf(`%s key, %s/%s ConfigMap`, f.key, f.cmNamespace, f.cmName)
}

// Read reads a file from the ConfigMap.
func (f CMFile) Read() ([]byte, error) {
	cm, err := f.client.ConfigMaps(f.cmNamespace).Get(f.cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data, ok := cm.Data[f.key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	return []byte(data), nil
}

// Write stores data to the ConfigMap. If provided ConfigMap doesn't exist it creates
// the new one.
func (f CMFile) Write(data []byte) error {
	_, err := f.client.ConfigMaps(f.cmNamespace).Get(f.cmName, metav1.GetOptions{})
	if err != nil {
		// ensure ConfigMap has created
		// TODO: do we need to ensure this? (similar to os.O_CREATE param for file)
		if f.cmCreate && apierrors.IsNotFound(err) {
			_, err = f.client.ConfigMaps(f.cmNamespace).Create(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      f.cmName,
					Namespace: f.cmNamespace,
				},
				Data: map[string]string{
					f.key: string(data),
				},
			})
		}
		return err
	}

	_, err = f.client.ConfigMaps(f.cmNamespace).Patch(
		f.cmName,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"data":{%q:%q}}`, f.key, string(data))),
	)
	return errors.Wrap(err, "patch configMap")
}
