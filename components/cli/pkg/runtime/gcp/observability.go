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

func CreateObservabilityConfigMaps() error {
	for _, confMap := range buildObservabilityConfigMaps() {
		err := kubernetes.CreateConfigMapWithNamespace(confMap.Name, confMap.Path, "cellery-system")
		if err != nil {
			return err
		}
	}
	return nil
}

func buildObservabilityConfigMaps() []ConfigMap {
	base := buildArtifactsPath(runtime.Observability)
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

//func AddObservability() error {
//	for _, v := range buildObservabilityYamlPaths() {
//		err := kubernetes.ApplyFile(v)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func buildObservabilityYamlPaths() []string {
	base := buildArtifactsPath(runtime.Observability)
	return []string{
		filepath.Join(base, "sp", "sp-worker.yaml"),
		filepath.Join(base, "portal", "observability-portal.yaml"),
		filepath.Join(base, "prometheus", "k8s-metrics-prometheus.yaml"),
		filepath.Join(base, "grafana", "k8s-metrics-grafana.yaml"),
		filepath.Join(base, "observability-agent", "telemetry-agent.yaml"),
		filepath.Join(base, "observability-agent", "tracing-agent.yaml"),
	}
}

func AddObservability(celleryValues runtime.CelleryRuntimeVals) error {
	chartName := "cellery-runtime"
	celleryVals, errCelVals := util.GetHelmChartDefaultValues(chartName)
	if errCelVals != nil {
		err := yaml.Unmarshal([]byte(celleryVals), &celleryValues)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}
	celleryValues.Observability.Enabled = true
	celleryYamls, errcon := yaml.Marshal(&celleryValues)
	if errcon != nil {
		log.Printf("error: %v", errcon)
	}
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-system",
		"delete", string(celleryYamls)); err != nil {
		return fmt.Errorf("error installing ingress controller: %v", err)
	}
	return nil
}