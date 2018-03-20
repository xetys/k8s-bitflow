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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"encoding/json"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
)

// genConfigCmd represents the genConfig command
var genConfigCmd = &cobra.Command{
	Use:   "gen-config",
	Short: "generates a JSON configuration for bitflow4j-anomaly-detector",
	Run:   RunGenConfig,
}

type ConfigObject struct {
	Children []string `json:"children,omitempty"`
	Vars     map[string]string `json:"vars,omitempty"`
	Hosts    []string `json:"hosts,omitempty"`
}

func RunGenConfig(cmd *cobra.Command, args []string) {
	outputFile, _ := cmd.Flags().GetString("output")
	print := outputFile == ""
	// create the clientset
	clientset, err := K8SClient()
	if err != nil {
		panic(err.Error())
	}

	bfConfig, err := GetClusterConfig(clientset)
	if err != nil {
		fmt.Errorf(err.Error())
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

func GetClusterConfig(clientset *kubernetes.Clientset) (map[string]ConfigObject, error) {
	bfConfig := make(map[string]ConfigObject)
	nodeNames := []string{}
	nodeChildren := make(map[string][]string)
	nodes, err := clientset.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		return nil, err
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
		return nil, err
	}
	for _, pod := range podList.Items {
		nodeName := pod.Spec.NodeName
		containerId := pod.Status.ContainerStatuses[0].ContainerID[9:]
		nodeChildren[nodeName] = append(nodeChildren[nodeName], containerId)
		bfConfig[containerId] = ConfigObject{Hosts: []string{pod.Name}}
		// fmt.Printf("container %s in namespace %s on node %s has id %s\n", pod.Name, pod.Namespace, pod.Spec.NodeName, pod.Status.ContainerStatuses[0].ContainerID[9:])
	}
	for nodeName, children := range nodeChildren {
		bfConfig[nodeName+"_vms"] = ConfigObject{Children: children}
	}
	return bfConfig, nil
}

func init() {
	rootCmd.AddCommand(genConfigCmd)
	genConfigCmd.Flags().StringP("output", "o", "", "output file. If empty, the config is printed to STDOUT")
}

// api call for bitflow
// POST /api/proc-children/<pod-name>?regex=<container-id>
// DELETE /api/proc-children