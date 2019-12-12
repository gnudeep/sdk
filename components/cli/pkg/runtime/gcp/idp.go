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

	"cellery.io/cellery/components/cli/pkg/kubernetes"
	"cellery.io/cellery/components/cli/pkg/runtime"
)

func CreateIdpConfigMaps() error {
	for _, confMap := range buildIdpConfigMaps() {
		err := kubernetes.CreateConfigMapWithNamespace(confMap.Name, confMap.Path, "cellery-system")
		if err != nil {
			return err
		}
	}
	return nil
}

//func CreateIdp() error {
//	for _, v := range buildIdpYamlPaths() {
//		err := kubernetes.ApplyFileWithNamespace(v, "cellery-system")
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func buildIdpConfigMaps() []ConfigMap {
	base := buildArtifactsPath(runtime.IdentityProvider)
	return []ConfigMap{
		{"identity-server-conf", filepath.Join(base, "conf")},
		{"identity-server-conf-datasources", filepath.Join(base, "conf", "datasources")},
		{"identity-server-conf-identity", filepath.Join(base, "conf", "identity")},
		{"identity-server-tomcat", filepath.Join(base, "conf", "tomcat")},
	}
}

func buildIdpYamlPaths() []string {
	base := buildArtifactsPath(runtime.IdentityProvider)
	return []string{
		filepath.Join(base, "global-idp.yaml"),
	}
}

func CreateIdp(celleryValues runtime.CelleryRuntimeVals) error {
	log.Printf("Deploying cellery runtime using cellery-runtime chart")
	chartName := "cellery-runtime"
	celleryVals, errCelVals := util.GetHelmChartDefaultValues(chartName)
	if errCelVals != nil {
		log.Fatalf("error: %v", errCelVals)
	}
	err := yaml.Unmarshal([]byte(celleryVals), &celleryValues)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	celleryValues.Idp.Enabled = true

	celleryYamls, errcon := yaml.Marshal(&celleryValues)
	if errcon != nil {
		log.Fatalf("error: %v", errcon)
	}
	//log.Printf(string(celleryYamls))
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-system",
		"apply", string(celleryYamls)); err != nil {
		fmt.Errorf("error installing ingress controller: %v", err)
	}
	return nil
}