package utils

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)


type KauloudConfig struct {
	ResourceDescriber struct {
		SkipCompressibleResource bool `yaml:"skipCompressibleResource"`
	} `yaml:"resourceDescriber"`
	VirtManagerConfig struct {
		GPRC struct {
			Address string `yaml:"address"`
			Port	string `yaml:"port"`
		} `yaml:"gRPC"`
		Namespace 	string `yaml:"namespace"`
		BaseImageNamespace string `yaml:"baseImageNamespace"`
		StorageClassName string `yaml:"storageClassName"`
		VmPreset struct {
			RunStrategy		string `yaml:"runStrategy"`
			DomainCpuCores		uint32 `yaml:"domainCpuCores"`
			Resources struct {
				Cpu string `yaml:"cpu"`
				Memory string `yaml:"memory"`
				Storage string `yaml:"storage"`
				Gpu struct {
					Enabled bool `yaml:"enabled"`
					GpuName string `yaml:"gpuName"`
				} `yaml:"gpu"`
			} `yaml:"resources"`
			PresetBaseImageName string `yaml:"presetBaseImageName"`
			Labels              map[string]string `yaml:"labels"`
		} `yaml:"vmPreset"`
		GracePeriodSeconds int `yaml:"gracePeriodSeconds"`
	} `yaml:"virtManager"`
}

func GetKauloudConfigFromLocalYamlFile(path string) (*KauloudConfig, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := &KauloudConfig{}
	err = yaml.Unmarshal(buf, config)
	if err != nil {
		return nil, err
	}
	return config, err
}

func DefaultKubeConfigPath() (string, error) {
	// TODO :: 1. make to build kube config from cli and yaml configuration.

	if home := homedir.HomeDir(); home != "" {
		config := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(config) ; err != nil {
			return "", err
		} else {
			return config, nil
		}
	} else {
		return "", errors.New("kube config path error")
	}
}
