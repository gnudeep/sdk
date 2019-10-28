/*
 * Copyright (c) 2019 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
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
)

func RenderHelmChart(releaseName string, chartNamespace string, chartPath string, chartValues string) string {
	if chartNamespace == "" {
		chartNamespace = "default"
	}
	//chartValues :=""
	//if valueFilePath != "" {
	//	chartValues = ReadValueFile(valueFilePath)
	//}
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
			//fmt.Println(v.Name)
			//fmt.Println(v.Content)
			//fmt.Println("---")
			tmplString = tmplString + "\n---\n" + v.Content
		}
	}
	return tmplString
}

func ApplyHelmTemplates(tmplString string) {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("Command Build", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, tmplString)
	}()

	out, err := cmd.CombinedOutput()
	WriteErrorLog(string(out))
	//fmt.Println("Commnd output :", string(out))

	if err != nil {
		log.Fatal("Command execution", err)
	}
}

func GetChartFromDir(dir string) (*chart.Chart, error) {

	helmChart, err := chartutil.LoadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "loading chart archive")
	}
	return helmChart, err
}

//Read custom value file
//func ReadValueFile(filePath string) string {
//	data, err := ioutil.ReadFile(filePath)
//	if err != nil {
//		log.Fatal(err)
//	}
//	//log.Println("Values",string(data))
//	return string(data)
//}

func GetHelmChartsValues(chartName string, chartPath string) string {
	data, err := ioutil.ReadFile(chartPath + "/" + chartName + "/values.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Values", string(data))
	return string(data)

}

func GetHelmChartsCustomValues(chartName string, chartPath string, valueFile string) string {
	data, err := ioutil.ReadFile(filepath.Join(chartPath, chartName, valueFile))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Values", string(data))
	return string(data)

}

func WriteErrorLog(templateApplyError string) {

	fileData := []byte(templateApplyError)
	f, err := os.OpenFile("cellery-tmpl-gen-error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(fileData)); err != nil {
		f.Close() // ignore error; Write error takes precedence
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}
