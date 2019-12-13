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

package setup

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"path/filepath"
	"time"

	"github.com/manifoldco/promptui"

	"cellery.io/cellery/components/cli/cli"
	"cellery.io/cellery/components/cli/pkg/constants"
	gcpPlatform "cellery.io/cellery/components/cli/pkg/gcp"
	"cellery.io/cellery/components/cli/pkg/kubernetes"
	"cellery.io/cellery/components/cli/pkg/runtime"
	"cellery.io/cellery/components/cli/pkg/runtime/gcp"
	"cellery.io/cellery/components/cli/pkg/util"
)

var accountName string

func RunSetupCreateGcp(isCompleteSetup bool) error {
	util.CopyK8sArtifacts(util.UserHomeCelleryDir())
	util.CopyHelmArtifacts(util.UserHomeCelleryDir())
	gcpSpinner := util.StartNewSpinner("Creating Cellery runtime on celleryGcp")
	platform, err := gcpPlatform.NewGcp()
	if err != nil {
		return fmt.Errorf("failed to initialize celleryGcp platform, %v", err)
	}
	err = platform.Create()
	if err != nil {
		gcpSpinner.Stop(false)
	}
	gcpSpinner.SetNewAction("Installing cellery runtime")
	if isCompleteSetup {
		createCompleteGcpRuntime(platform)
	} else {
		createMinimalGcpRuntime(platform)
	}
	runtime.WaitFor(true, false)
	return nil
}

func createGcp(cli cli.Cli) error {
	var isCompleteSelected = false
	cellTemplate := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U000027A4 {{ .| bold }}",
		Inactive: "  {{ . | faint }}",
		Help:     util.Faint("[Use arrow keys]"),
	}

	cellPrompt := promptui.Select{
		Label:     util.YellowBold("?") + " Select the type of runtime",
		Items:     []string{constants.BASIC, constants.COMPLETE, setupBack},
		Templates: cellTemplate,
	}
	_, value, err := cellPrompt.Run()
	if err != nil {
		return fmt.Errorf("failed to select an option: %v", err)
	}
	if value == setupBack {
		createEnvironment(cli)
		return nil
	}
	if value == constants.COMPLETE {
		isCompleteSelected = true
	}
	RunSetupCreateGcp(isCompleteSelected)
	return nil
}

func createMinimalGcpRuntime(platform *gcpPlatform.Gcp) {
	// Deploy cellery runtime
	deployMinimalCelleryRuntime(platform)
	util.RemoveDir(filepath.Join(util.UserHomeCelleryDir(), constants.K8sArtifacts))
}

func createCompleteGcpRuntime(platform *gcpPlatform.Gcp) error {
	// Deploy cellery runtime
	deployCompleteCelleryRuntime(platform)
	util.RemoveDir(filepath.Join(util.UserHomeCelleryDir(), constants.K8sArtifacts))
	return nil
}

func createControllerx() error {
	// Give permission to the user
	if err := kubernetes.CreateClusterRoleBinding("cluster-admin", accountName); err != nil {
		return fmt.Errorf("error creating cluster role binding, %v", err)
	}
	// Setup Cellery namespace
	if err := runtime.CreateCelleryNameSpace(); err != nil {
		return fmt.Errorf("error creating cellery namespace, %v", err)
	}
	// Apply Istio CRDs
	if err := runtime.ApplyIstioCrds(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
		return fmt.Errorf("error applying istio crds, %v", err)
	}
	// sleep for few seconds - this is to make sure that the CRDs are properly applied
	time.Sleep(20 * time.Second)
	// Enabling Istio injection
	if err := kubernetes.ApplyLable("namespace", "default", "istio-injection=enabled",
		false); err != nil {
		return err
	}
	// Install istio
	if err := runtime.InstallIstio(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
		return err
	}
	// Install knative serving
	if err := runtime.InstallKnativeServing(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
		return err
	}
	// Apply controller CRDs
	if err := runtime.InstallController(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
		return err
	}
	return nil
}

func createController() error {
	if err := kubernetes.CreateClusterRoleBinding("cluster-admin", accountName); err != nil {
		return fmt.Errorf("error creating cluster role binding, %v", err)
	}
	// Setup Cellery namespace
	if err := util.CreateNameSpace("cellery-system"); err != nil {
		return fmt.Errorf("error creating cellery namespace, %v", err)
	}
	// Setup istio-system namespace
	if err := util.CreateNameSpace("istio-system"); err != nil {
		return fmt.Errorf("error creating cellery namespace, %v", err)
	}
	//Install istio crds and components.
	log.Printf("Deploying istio CRDs using istio-init chart")
	if err := util.ApplyHelmChartWithDefaultValues("istio-init", "istio-system"); err != nil {
		return fmt.Errorf("error installing istio crds: %v", err)
	}
	// sleep for few seconds - this is to make sure that the CRDs are properly applied
	time.Sleep(20 * time.Second)
	// Enabling Istio injection
	if err := kubernetes.ApplyLable("namespace", "default", "istio-injection=enabled",
		false); err != nil {
		return err
	}
	log.Printf("Deploying istio system using istio chart")
	if err := util.ApplyHelmChartWithDefaultValues("istio", "istio-system"); err != nil {
		return fmt.Errorf("error installing istio : %v", err)
	}
	log.Printf("Deploying knative system using knative-crd chart")
	if err := util.ApplyHelmChartWithDefaultValues("knative-crd", "default"); err != nil {
		return fmt.Errorf("error installing knative crds: %v", err)
	}
	return nil
}

func deployMinimalCelleryRuntime(platform *gcpPlatform.Gcp) error {
	celleryValues := runtime.CelleryRuntimeVals{}
	chartName := "cellery-runtime"
	celleryVals, errCelVals := util.GetHelmChartDefaultValues(chartName)
	if errCelVals != nil {
		log.Fatalf("error: %v", errCelVals)
	}
	err := yaml.Unmarshal([]byte(celleryVals), &celleryValues)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	celleryValues.Global.CelleryRuntime.Db.Hostname = platform.SqlHostName
	celleryValues.Global.CelleryRuntime.Db.CarbonDb.Username = platform.SqlCredential.SqlUserName
	celleryValues.Global.CelleryRuntime.Db.CarbonDb.Password = platform.SqlCredential.SqlPassword
	errorDeployingCelleryRuntime := "Error deploying cellery runtime"

	createController()
	//createAllDeploymentArtifacts()
	createIdpGcp(celleryValues, errorDeployingCelleryRuntime)
	createNGinx(errorDeployingCelleryRuntime)

	return nil
}

func deployCompleteCelleryRuntime(platform *gcpPlatform.Gcp) {
	celleryValues := runtime.CelleryRuntimeVals{}
	celleryValues.Global.CelleryRuntime.Db.CarbonDb.Username = platform.SqlCredential.SqlUserName
	celleryValues.Global.CelleryRuntime.Db.CarbonDb.Password = platform.SqlCredential.SqlPassword
	errorDeployingCelleryRuntime := "Error deploying cellery runtime"

	createController()
	createAllDeploymentArtifacts()

	//Create gateway deployment and the service
	if err := gcp.AddApim(celleryValues); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}

	// Create observability
	if err := gcp.AddObservability(celleryValues); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}
	//Create NGinx
	createNGinx(errorDeployingCelleryRuntime)
}

func createIdpGcp(celleryValues runtime.CelleryRuntimeVals, errorDeployingCelleryRuntime string) {
	//Create IDP deployment and the service
	if err := gcp.CreateIdp(celleryValues); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}


}

func createNGinx(errorMessage string) {
	//Install nginx-ingress for control plane ingress
	if err := gcp.InstallNginx(); err != nil {
		util.ExitWithErrorMessage(errorMessage, err)
	}

}

func createAllDeploymentArtifacts() {
	errorDeployingCelleryRuntime := "Error deploying cellery runtime"

	// Create apim NFS volumes and volume claims
	if err := gcp.CreatePersistentVolume(); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}
	// Create the gw config maps
	if err := gcp.CreateGlobalGatewayConfigMaps(); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}
	// Create Observability configmaps
	if err := gcp.CreateObservabilityConfigMaps(); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}
	// Create the IDP config maps
	if err := gcp.CreateIdpConfigMaps(); err != nil {
		util.ExitWithErrorMessage(errorDeployingCelleryRuntime, err)
	}
}
