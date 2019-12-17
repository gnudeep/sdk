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
	"cellery.io/cellery/components/cli/pkg/util"
	"path/filepath"

	"cellery.io/cellery/components/cli/pkg/kubernetes"
)

func (runtime *CelleryRuntime) AddIdp(db MysqlDb) error {
	//for _, v := range buildIdpYamlPaths(runtime.artifactsPath) {
	//	err := kubernetes.ApplyFileWithNamespace(v, "cellery-system")
	//	if err != nil {
	//		return err
	//	}
	//}
	runtime.UnmarshalHelmValues("cellery-runtime")
	runtime.celleryRuntimeVals.Idp.Enabled = true
	if runtime.IsGcpRuntime() {
		runtime.celleryRuntimeVals.Global.CelleryRuntime.Db.Hostname = db.DbHostName
		runtime.celleryRuntimeVals.Global.CelleryRuntime.Db.CarbonDb.Username = db.DbUserName
		runtime.celleryRuntimeVals.Global.CelleryRuntime.Db.CarbonDb.Password = db.DbPassword
	}
	runtime.MarshalHelmValues("cellery-runtime")
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-system",
		"apply", runtime.celleryRuntimeYaml); err != nil {
		return err
	}
	return nil
}

func deleteIdp(artifactsPath string) error {
	for _, v := range buildIdpYamlPaths(artifactsPath) {
		err := kubernetes.DeleteFileWithNamespace(v, "cellery-system")
		if err != nil {
			return err
		}
	}
	return nil
}

func (runtime *CelleryRuntime) DeleteIdp() error {
	runtime.UnmarshalHelmValues("cellery-runtime")
	runtime.celleryRuntimeVals.Idp.Enabled = true
	runtime.MarshalHelmValues("cellery-runtime")
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-system",
		"delete", runtime.celleryRuntimeYaml); err != nil {
		return err
	}
	return nil
}

func createIdpConfigMaps(artifactsPath string) error {
	for _, confMap := range buildIdpConfigMaps(artifactsPath) {
		err := kubernetes.CreateConfigMapWithNamespace(confMap.Name, confMap.Path, "cellery-system")
		if err != nil {
			return err
		}
	}
	return nil
}

func buildIdpYamlPaths(artifactsPath string) []string {
	base := buildArtifactsPath(IdentityProvider, artifactsPath)
	return []string{
		filepath.Join(base, "global-idp.yaml"),
	}
}

func buildIdpConfigMaps(artifactsPath string) []ConfigMap {
	base := buildArtifactsPath(IdentityProvider, artifactsPath)
	return []ConfigMap{
		{"identity-server-conf", filepath.Join(base, "conf")},
		{"identity-server-conf-datasources", filepath.Join(base, "conf", "datasources")},
		{"identity-server-conf-identity", filepath.Join(base, "conf", "identity")},
		{"identity-server-tomcat", filepath.Join(base, "conf", "tomcat")},
	}
}
