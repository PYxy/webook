package myEtcd

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type LocalServer struct {
	GRPC map[string]ServicePart `yaml:"grpc"`
}

type ServicePart struct {
	Nodes []NodePort `yaml:"nodes"`
}

type NodePort struct {
	Address string         `yaml:"address"`
	Labels  map[string]any `yaml:"labels"`
}

func ReadYaml(target *LocalServer, filePath string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(content, &target)
	if err != nil {
		return err
	}

	return nil
}

func SaveYaml(src *LocalServer, filePath string) error {
	fmt.Println("保存")
	data, err := yaml.Marshal(src) // 第二个表示每行的前缀，这里不用，第三个是缩进符号，这里用tab
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, data, 0777)
	if err != nil {
		return err
	}
	return nil
}
