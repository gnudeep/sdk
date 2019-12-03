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

package runtime

import (
	"cellery.io/cellery/components/cli/pkg/kubernetes"
	"cellery.io/cellery/components/cli/pkg/util"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"path/filepath"
	"strings"
)

func addObservability(artifactsPath string) error {
	for _, v := range buildObservabilityYamlPaths(artifactsPath) {
		err := kubernetes.ApplyFile(v)
		if err != nil {
			return err
		}
	}
	return nil
}

//func deleteObservability(artifactsPath string) error {
//	for _, v := range buildObservabilityYamlPaths(artifactsPath) {
//		err := kubernetes.DeleteFile(v)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
func IsObservabilityEnabled() (bool, error) {
	enabled := true
	_, err := kubernetes.GetDeployment("cellery-system", "wso2sp-worker")
	if err != nil {
		if strings.Contains(err.Error(), "No resources found") ||
			strings.Contains(err.Error(), "Error from server (NotFound)") {
			enabled = false
		} else {
			return enabled, fmt.Errorf("error checking if observability is enabled")
		}
	}
	return enabled, nil
}

func CreateObservabilityConfigMaps(artifactsPath string) error {
	for _, confMap := range buildObservabilityConfigMaps(artifactsPath) {
		err := kubernetes.CreateConfigMapWithNamespace(confMap.Name, confMap.Path, "cellery-system")
		if err != nil {
			return err
		}
	}
	return nil
}

func buildObservabilityYamlPaths(artifactsPath string) []string {
	base := buildArtifactsPath(Observability, artifactsPath)
	return []string{
		filepath.Join(base, "sp", "sp-worker.yaml"),
		filepath.Join(base, "portal", "observability-portal.yaml"),
		filepath.Join(base, "prometheus", "k8s-metrics-prometheus.yaml"),
		filepath.Join(base, "grafana", "k8s-metrics-grafana.yaml"),
		filepath.Join(base, "mixer-adapter", "mixer-adapter.yaml"),
	}
}

func buildObservabilityConfigMaps(artifactsPath string) []ConfigMap {
	base := buildArtifactsPath(Observability, artifactsPath)
	return []ConfigMap{
		{"sp-worker-siddhi", filepath.Join(base, "siddhi")},
		{"sp-worker-conf", filepath.Join(base, "sp", "conf")},
		{"observability-portal-config", filepath.Join(base, "node-server", "config")},
		{"k8s-metrics-prometheus-conf", filepath.Join(base, "prometheus", "config")},
		{"k8s-metrics-grafana-conf", filepath.Join(base, "grafana", "config")},
		{"k8s-metrics-grafana-datasources", filepath.Join(base, "grafana", "datasources")},
		{"k8s-metrics-grafana-dashboards", filepath.Join(base, "grafana", "dashboards")},
		{"k8s-metrics-grafana-dashboards-default", filepath.Join(base, "grafana", "dashboards", "default")},
	}
}

func deleteObservability() error {
	celleryValues := CelleryRuntimeVals{}
	chartName := "cellery-runtime"
	celleryVals, errCelVals := util.GetHelmChartDefaultValues(chartName)
	if errCelVals != nil {
		err := yaml.Unmarshal([]byte(celleryVals), &celleryValues)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}
	celleryValues.Idp.Enabled = true
	celleryYamls, errcon := yaml.Marshal(&celleryValues)
	if errcon != nil {
		log.Printf("error: %v", errcon)
	}
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-runtime", "delete", string(celleryYamls)); err != nil {
		return fmt.Errorf("error installing ingress controller: %v", err)
	}
	return nil
}