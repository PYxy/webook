package myEtcd

import (
	"fmt"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteYaml(t *testing.T) {
	var tmp map[string]any
	tmp = make(map[string]any, 2)
	tmp["age"] = 10
	tmp["name"] = "小白"
	ls := LocalServer{GRPC: map[string]ServicePart{

		"service/user": {
			Nodes: []NodePort{
				{
					Address: "127.0.0.1:8989",
					Labels:  tmp,
				},
				{
					Address: "127.0.0.1:8988",
					Labels: map[string]any{
						"weight": 7,
					},
				},
			},
		},
		"service/good": {
			Nodes: []NodePort{
				{
					Address: "127.0.0.1:8989",
					Labels:  tmp,
				},
				{
					Address: "127.0.0.1:8988",
					Labels: map[string]any{
						"weight": 7,
					},
				},
			},
		},
	}}
	data, err := yaml.Marshal(ls) // 第二个表示每行的前缀，这里不用，第三个是缩进符号，这里用tab
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile("./grpc.yaml", data, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func TestReadYaml(t *testing.T) {
	content, err := ioutil.ReadFile("./grpc.yaml")
	if err != nil {
		panic(err)
	}
	var ls *LocalServer

	ls = &LocalServer{}
	err = yaml.Unmarshal(content, &ls)
	if err != nil {
		panic(err)
	}

	for key, val := range ls.GRPC {
		fmt.Println("====================")
		fmt.Println("服务名:", key)
		for _, node := range val.Nodes {
			fmt.Println(node.Address, "  ,", node.Labels)
			fmt.Println("++++++++++++++++++++++++++++++++")
		}

	}

}
