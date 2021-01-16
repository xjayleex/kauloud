package tmp

import (
	"github.com/pkg/errors"
	_ "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type KubeClient struct {
	clientset *kubernetes.Clientset
	kubeconfig *string
}

func NewKubeConfig () (*string, error) {
	// TODO :: 1. make to build kube config from cli and yaml configuration.

	if home := homedir.HomeDir(); home != "" {
		kc := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kc) ; err != nil {
			return nil, err
		} else {
			return &kc, nil
		}
	} else {
		return nil, errors.New("kube config path error")
	}
}

func (kc *KubeClient) Clientset() *kubernetes.Clientset{
	return kc.clientset
}

func NewKubeClient (kubeconfig *string) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		clientset: clientset,
		kubeconfig: kubeconfig,
	}, nil
}