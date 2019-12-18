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
 *
 */
package util

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"

	"cellery.io/cellery/components/cli/pkg/constants"
)

//Generate k8s artifacts using helm templates.
func RenderHelmChart(releaseName string, chartNamespace string, chartPath string, chartValues string) string {
	if chartNamespace == "" {
		chartNamespace = "default"
	}
	celleryChart, err := GetChartFromDir(chartPath)
	if err != nil {
		log.Fatal(err)
	}
	releaseOptions := chartutil.ReleaseOptions{Name: releaseName, Namespace: chartNamespace, IsInstall: true}
	renderOptions := renderutil.Options{ReleaseOptions: releaseOptions}
	manifestList := make(map[string]string)
	if chartValues == "" {
		manifestList, err = renderutil.Render(celleryChart, &chart.Config{}, renderOptions)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		manifestList, err = renderutil.Render(celleryChart, &chart.Config{Raw: chartValues}, renderOptions)
		if err != nil {
			log.Fatal(err)
		}
	}
	tmplString := ""
	manifests := manifest.SplitManifests(manifestList)
	for _, v := range manifests {
		if !strings.Contains(v.Name, "NOTES.txt") {
			tmplString = tmplString + "\n---\n" + v.Content
		}
	}
	return tmplString
}

//Read helm chart from the filesystem.
func GetChartFromDir(dir string) (*chart.Chart, error) {
	helmChart, err := chartutil.LoadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "loading chart templates.")
	}
	return helmChart, err
}

//Read helm chart value file.
//Chart path differs based on the installed OS.
func GetHelmChartsCustomValues(chartName string, chartPath string, valueFile string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(chartPath, chartName, valueFile))
	if err != nil {
		log.Printf("Value file access error %s \n", err)
		return "", err
	}
	return string(data), nil
}

func GetHelmChartDefaultValues(chartName string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(filepath.Join(CelleryInstallationDir(), constants.HelmCarts), chartName, "values.yaml"))
	if err != nil {
		log.Printf("Value file access error %s \n", err)
		return "", err
	}
	return string(data), nil
}

//Apply generated k8s artifacts.
func ApplyHelmTemplates(tmplString string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("kubectl command build error %s \n", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, tmplString)
	}()
	out, err := cmd.CombinedOutput()
	log.Printf("kubectl apply commnd output : %s", string(out))

	if err != nil {
		log.Printf("kubectl command execution error %s \n", err)
		return err
	}
	return nil
}

func ApplyK8sResource(operation string, tmplString string, namespace string) error {
	cmd := exec.Command("kubectl", operation, "-f", "-", "-n", namespace)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("kubectl command build", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, tmplString)
	}()
	out, err := cmd.CombinedOutput()
	log.Printf("kubectl apply commnd output : %s", string(out))
	if err != nil {
		log.Print("kubectl command execution", err)
		return err
	}
	return nil
}

//Apply helm chart with default values
func ApplyHelmChartWithDefaultValues(chartName string, namespace string) error {
	chartTemplate := RenderHelmChart(chartName, namespace, filepath.Join(UserHomeCelleryDir(), constants.HelmCarts, chartName), "")
	err := ApplyHelmTemplates(chartTemplate)
	if err != nil {
		return err
	}
	return nil
}

func ApplyHelmChartWithDefaultValuesCustomCmd(chartName string, namespace string, operation string) error {
	chartTemplate := RenderHelmChart(chartName, namespace, filepath.Join(UserHomeCelleryDir(), constants.HelmCarts, chartName), "")
	err := ApplyK8sResource(operation, chartTemplate, namespace)
	if err != nil {
		return err
	}
	return nil
}

func ApplyHelmChartWithCustomValues(chartName string, namespace string, operation string, values string) error {
	chartTemplate := RenderHelmChart(chartName, namespace, filepath.Join(UserHomeCelleryDir(), constants.HelmCarts, chartName), values)
	log.Printf(chartTemplate)
	err := ApplyK8sResource(operation, chartTemplate, namespace)
	if err != nil {
		return err
	}
	return nil
}

func CreateNameSpace(namespace string) error {
	var cmd *exec.Cmd
	cmd = exec.Command(
		constants.KubeCtl,
		"create",
		"ns",
		namespace)

	cmd.Stderr = os.Stderr
	return cmd.Run()
}
