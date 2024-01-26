package main

import (
	"flag"
	"fmt"

	"k8s.io/kubernetes/test/e2e_node/remote"
	"k8s.io/kubernetes/test/e2e_node/remote/gce"
)

func main() {
	flag.Parse()
	cfg := remote.Config{
		Images: []string{
			"cos-beta-109-17800-66-65",
		},
	}
	runner := gce.NewGCERunner2(cfg)
	err := runner.Validate()
	if err != nil {
		fmt.Println(err)
	}
	image, err := runner.GetGCEImage("", "", "k8s-infra-e2e-node-e2e-project")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(image)
	err = runner.RegisterGceHostIP("instance-1")
	if err != nil {
		fmt.Println(err)
	}
	//runner.DeleteGCEInstance("instance-2")
	data, err := runner.GetSerialOutput("instance-1")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data)
	var suite remote.TestSuite
	runner.StartTests(suite, "/tmp", nil)
}
