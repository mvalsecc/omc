/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gmeghnag/omc/cmd/helpers"
	"github.com/gmeghnag/omc/vars"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type ServicesItems struct {
	ApiVersion string            `json:"apiVersion"`
	Items      []*corev1.Service `json:"items"`
}

func GetServices(currentContextPath string, namespace string, resourceName string, allNamespacesFlag bool, outputFlag string, showLabels bool, jsonPathTemplate string, allResources bool) bool {
	_headers := []string{"namespace", "name", "type", "cluster-ip", "external-ip", "port(s)", "age", "selector"}
	var namespaces []string
	if allNamespacesFlag == true {
		namespace = "all"
		_namespaces, _ := ioutil.ReadDir(currentContextPath + "/namespaces/")
		for _, f := range _namespaces {
			namespaces = append(namespaces, f.Name())
		}
	} else {
		namespaces = append(namespaces, namespace)
	}

	var data [][]string
	var _ServicesList = ServicesItems{ApiVersion: "v1"}
	for _, _namespace := range namespaces {
		var _Items ServicesItems
		CurrentNamespacePath := currentContextPath + "/namespaces/" + _namespace
		_file, err := ioutil.ReadFile(CurrentNamespacePath + "/core/services.yaml")
		if err != nil && !allNamespacesFlag {
			fmt.Fprintln(os.Stderr, "No resources found in "+_namespace+" namespace.")
			os.Exit(1)
		}
		if err := yaml.Unmarshal([]byte(_file), &_Items); err != nil {
			fmt.Fprintln(os.Stderr, "Error when trying to unmarshal file "+CurrentNamespacePath+"/core/services.yaml")
			os.Exit(1)
		}

		for _, Service := range _Items.Items {
			labels := helpers.ExtractLabels(Service.GetLabels())
			if !helpers.MatchLabels(labels, vars.LabelSelectorStringVar) {
				continue
			}

			if resourceName != "" && resourceName != Service.Name {
				continue
			}
			if outputFlag == "name" {
				_ServicesList.Items = append(_ServicesList.Items, Service)
				fmt.Println("service/" + Service.Name)
				continue
			}

			if outputFlag == "yaml" {
				_ServicesList.Items = append(_ServicesList.Items, Service)
				continue
			}

			if outputFlag == "json" {
				_ServicesList.Items = append(_ServicesList.Items, Service)
				continue
			}

			if strings.HasPrefix(outputFlag, "jsonpath=") {
				_ServicesList.Items = append(_ServicesList.Items, Service)
				continue
			}

			//name
			ServiceName := Service.Name
			if allResources {
				ServiceName = "service/" + ServiceName
			}

			//age
			age := helpers.GetAge(CurrentNamespacePath+"/core/services.yaml", Service.GetCreationTimestamp())

			//cluster-ip
			ClusterIp := "<none>"
			if Service.Spec.ClusterIP != "" {
				ClusterIp = Service.Spec.ClusterIP
			}
			//external-ip
			externalIp := "<none>"
			if string(Service.Spec.Type) == "ExternalName" {
				externalIp = Service.Spec.ExternalName
			}
			if string(Service.Spec.Type) == "ClusterIp" {
				externalIp = Service.Spec.ClusterIP
			}
			if string(Service.Spec.Type) == "LoadBalancer" {
				externalIp = ""
				for _, p := range Service.Status.LoadBalancer.Ingress {
					externalIp += fmt.Sprint(p.Hostname) + ","
				}
				externalIp = strings.TrimRight(externalIp, ",")
			}
			//ports
			ports := ""
			for _, p := range Service.Spec.Ports {
				ports += fmt.Sprint(p.Port) + "/" + string(p.Protocol) + ","
			}
			if ports == "" {
				ports = "<none>"
			} else {
				ports = strings.TrimRight(ports, ",")
			}
			//selector
			selector := ""
			for k, v := range Service.Spec.Selector {
				selector += k + "=" + v + ","
			}
			if selector == "" {
				selector = "<none>"
			} else {
				selector = strings.TrimRight(selector, ",")
			}

			_list := []string{Service.Namespace, ServiceName, string(Service.Spec.Type), ClusterIp, externalIp, ports, age, selector}
			data = helpers.GetData(data, allNamespacesFlag, showLabels, labels, outputFlag, 7, _list)

			if resourceName != "" && resourceName == ServiceName {
				break
			}
		}
		if namespace != "" && _namespace == namespace {
			break
		}
	}

	if (outputFlag == "" || outputFlag == "wide") && len(data) == 0 {
		if !allResources {
			fmt.Println("No resources found in " + namespace + " namespace.")
		}
		return true
	}

	var headers []string
	if outputFlag == "" {
		if allNamespacesFlag == true {
			headers = _headers[0:7]
		} else {
			headers = _headers[1:7]
		}
		if showLabels {
			headers = append(headers, "labels")
		}
		helpers.PrintTable(headers, data)
		return false
	}
	if outputFlag == "wide" {
		if allNamespacesFlag == true {
			headers = _headers
		} else {
			headers = _headers[1:]
		}
		if showLabels {
			headers = append(headers, "labels")
		}
		helpers.PrintTable(headers, data)
		return false
	}

	if len(_ServicesList.Items) == 0 {
		if !allResources {
			fmt.Println("No resources found in " + namespace + " namespace.")
		}
		return true
	}

	var resource interface{}
	if resourceName != "" {
		resource = _ServicesList.Items[0]
	} else {
		resource = _ServicesList
	}

	if outputFlag == "yaml" {
		y, _ := yaml.Marshal(resource)
		fmt.Println(string(y))
	}
	if outputFlag == "json" {
		j, _ := json.MarshalIndent(resource, "", "  ")
		fmt.Println(string(j))
	}
	if strings.HasPrefix(outputFlag, "jsonpath=") {
		helpers.ExecuteJsonPath(resource, jsonPathTemplate)
	}
	return false
}

var Service = &cobra.Command{
	Use:     "service",
	Aliases: []string{"services", "svc"},
	Hidden:  true,
	Run: func(cmd *cobra.Command, args []string) {
		resourceName := ""
		if len(args) == 1 {
			resourceName = args[0]
		}
		jsonPathTemplate := helpers.GetJsonTemplate(vars.OutputStringVar)
		GetServices(vars.MustGatherRootPath, vars.Namespace, resourceName, vars.AllNamespaceBoolVar, vars.OutputStringVar, vars.ShowLabelsBoolVar, jsonPathTemplate, false)
	},
}
