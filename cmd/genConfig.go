// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"flag"
	"path/filepath"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"encoding/json"
	"io/ioutil"
)

// genConfigCmd represents the genConfig command
var genConfigCmd = &cobra.Command{
	Use:   "gen-config",
	Short: "generates a JSON configuration for bitflow4j-anomaly-detector",
	Run:   RunGenConfig,
}

type ConfigObject struct {
	Children []string `json:"children,omitempty"`
	Vars     []string `json:"vars,omitempty"`
	Hosts    []string `json:"hosts,omitempty"`
}

func RunGenConfig(cmd *cobra.Command, args []string) {
	outputFile, _ := cmd.Flags().GetString("output")
	print := outputFile == ""
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	bfConfig := make(map[string]ConfigObject)
	nodeNames := []string{}
	nodeChildren := make(map[string][]string)

	nodes, err := clientset.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		fmt.Errorf(err.Error())
	}

	for _, node := range nodes.Items {
		bfConfig[node.Name] = ConfigObject{
			Hosts: []string{node.Name},
		}

		nodeNames = append(nodeNames, node.Name)
		nodeChildren[node.Name] = []string{}
	}

	bfConfig["hypervisors"] = ConfigObject{Children: nodeNames}

	// render pods
	podList, err := clientset.CoreV1().Pods("").List(v1.ListOptions{})
	if err != nil {
		fmt.Errorf(err.Error())
	}

	for _, pod := range podList.Items {
		nodeName := pod.Spec.NodeName
		containerId := pod.Status.ContainerStatuses[0].ContainerID[9:]
		nodeChildren[nodeName] = append(nodeChildren[nodeName], containerId)
		// fmt.Printf("container %s in namespace %s on node %s has id %s\n", pod.Name, pod.Namespace, pod.Spec.NodeName, pod.Status.ContainerStatuses[0].ContainerID[9:])
	}

	for nodeName, children := range nodeChildren {
		bfConfig[nodeName + "_vms"] = ConfigObject{Children: children}
	}

	jsonBytes, err := json.MarshalIndent(bfConfig, "", "    ")
	if err != nil {
		fmt.Errorf(err.Error())
	}

	if print {
		fmt.Printf("%s", jsonBytes)
		fmt.Println()
	} else {
		ioutil.WriteFile(outputFile, jsonBytes, 0755)
	}
}

func init() {
	rootCmd.AddCommand(genConfigCmd)
	genConfigCmd.Flags().StringP("output", "o", "", "output file. If empty, the config is printed to STDOUT")
}
