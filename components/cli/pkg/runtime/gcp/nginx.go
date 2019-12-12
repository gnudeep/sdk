/*
 * Copyright (c) 2019 WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package gcp

import (
	"fmt"
	"github.com/cellery-io/sdk/components/cli/pkg/util"
	"gopkg.in/yaml.v2"
	"log"
	"path/filepath"

	"cellery.io/cellery/components/cli/pkg/runtime"
)

//func InstallNginx() error {
//	for _, file := range buildNginxYamlPaths() {
//		err := kubernetes.ApplyFile(file)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func buildNginxYamlPaths() []string {
	base := buildArtifactsPath(runtime.System)
	return []string{
		filepath.Join(base, "mandatory.yaml"),
		filepath.Join(base, "cloud-generic.yaml"),
	}
}

func InstallNginx() error {
	log.Printf("Deploying ingress controller Nodeport system using ingress-controller chart")
	ingressControllerVals := runtime.IngressController{}
	ingVals, errVal := util.GetHelmChartDefaultValues("ingress-controller")
	if errVal != nil {
		log.Fatalf("error: %v", errVal)
	}
	errYaml := yaml.Unmarshal([]byte(ingVals), &ingressControllerVals)
	if errYaml != nil {
		log.Fatalf("error: %v", errYaml)
	}
	ingressControllerVals.NginxIngress.Controller.Service.Type = "LoadBalancer"
	controllerYamls, errcon := yaml.Marshal(&ingressControllerVals)
	if errcon != nil {
		log.Fatalf("error: %v", errcon)
	}
	if err := util.ApplyHelmChartWithCustomValues("ingress-controller", "ingress-nginx",
		"apply", string(controllerYamls)); err != nil {
		fmt.Errorf("error installing ingress controller: %v", err)
	}
	return nil
}