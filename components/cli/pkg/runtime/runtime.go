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
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cellery.io/cellery/components/cli/pkg/constants"
	"cellery.io/cellery/components/cli/pkg/kubernetes"
	"cellery.io/cellery/components/cli/pkg/util"
)

type Selection int

const (
	NoChange Selection = iota
	Enable
	Disable
)

var isCompleteSetup = false

func SetCompleteSetup(completeSetup bool) {
	isCompleteSetup = completeSetup
}

type Runtime interface {
	Create() error
	IsComponentEnabled(component SystemComponent) (bool, error)
	SetArtifactsPath(artifactsPath string)
	SetPersistentVolume(isPersistentVolume bool)
	SetHasNfsStorage(hasNfsStorage bool)
	SetLoadBalancerIngressMode(isLoadBalancerIngressMode bool)
	SetNodePortIpAddress(nodePortIpAddress string)
	SetDb(db MysqlDb)
	SetNfs(nfs Nfs)
}

type CelleryRuntime struct {
	artifactsPath             string
	isPersistentVolume        bool
	hasNfsStorage             bool
	isLoadBalancerIngressMode bool
	nfs                       Nfs
	db                        MysqlDb
	nodePortIpAddress         string
}

// NewCelleryRuntime returns a CelleryRuntime instance.
func NewCelleryRuntime(opts ...func(*CelleryRuntime)) *CelleryRuntime {
	runtime := &CelleryRuntime{}
	for _, opt := range opts {
		opt(runtime)
	}
	return runtime
}

func (runtime *CelleryRuntime) SetArtifactsPath(artifactsPath string) {
	runtime.artifactsPath = artifactsPath
}

func (runtime *CelleryRuntime) SetPersistentVolume(isPersistentVolume bool) {
	runtime.isPersistentVolume = isPersistentVolume
}

func (runtime *CelleryRuntime) SetHasNfsStorage(hasNfsStorage bool) {
	runtime.hasNfsStorage = hasNfsStorage
}

func (runtime *CelleryRuntime) SetLoadBalancerIngressMode(isLoadBalancerIngressMode bool) {
	runtime.isLoadBalancerIngressMode = isLoadBalancerIngressMode
}

func (runtime *CelleryRuntime) SetNodePortIpAddress(nodePortIpAddress string) {
	runtime.nodePortIpAddress = nodePortIpAddress
}

func (runtime *CelleryRuntime) SetDb(db MysqlDb) {
	runtime.db = db
}

func (runtime *CelleryRuntime) SetNfs(nfs Nfs) {
	runtime.nfs = nfs
}

//func (runtime *CelleryRuntime) Create() error {
//	spinner := util.StartNewSpinner("Creating cellery runtime")
//	if runtime.isPersistentVolume && !runtime.hasNfsStorage {
//		createFoldersRequiredForMysqlPvc()
//		createFoldersRequiredForApimPvc()
//	}
//	dbHostName := constants.MysqlHostNameForExistingCluster
//	dbUserName := constants.CellerySqlUserName
//	dbPassword := constants.CellerySqlPassword
//	if runtime.hasNfsStorage {
//		dbHostName = runtime.db.DbHostName
//		dbUserName = runtime.db.DbUserName
//		dbPassword = runtime.db.DbPassword
//		updateNfsServerDetails(runtime.nfs.NfsServerIp, runtime.nfs.FileShare, runtime.artifactsPath)
//	}
//	if err := updateMysqlCredentials(dbUserName, dbPassword, dbHostName, runtime.artifactsPath); err != nil {
//		spinner.Stop(false)
//		return fmt.Errorf("error updating mysql credentials: %v", err)
//	}
//	if err := updateInitSql(dbUserName, dbPassword, runtime.artifactsPath); err != nil {
//		spinner.Stop(false)
//		return fmt.Errorf("error updating mysql init script: %v", err)
//	}
//
//	if runtime.isPersistentVolume && !IsGcpRuntime() {
//		nodeName, err := kubernetes.GetMasterNodeName()
//		if err != nil {
//			return fmt.Errorf("error getting master node name: %v", err)
//		}
//		if err := kubernetes.ApplyLable("nodes", nodeName, "disk=local", true); err != nil {
//			return fmt.Errorf("error applying master node lable: %v", err)
//		}
//	}
//	// Setup Cellery namespace
//	spinner.SetNewAction("Setting up cellery namespace")
//	if err := CreateCelleryNameSpace(); err != nil {
//		return fmt.Errorf("error creating cellery namespace: %v", err)
//	}
//
//	// Apply Istio CRDs
//	spinner.SetNewAction("Applying istio crds")
//	if err := ApplyIstioCrds(runtime.artifactsPath); err != nil {
//		return fmt.Errorf("error creating istio crds: %v", err)
//	}
//	// Apply nginx resources
//	spinner.SetNewAction("Creating ingress-nginx")
//	if err := installNginx(runtime.artifactsPath, runtime.isLoadBalancerIngressMode); err != nil {
//		return fmt.Errorf("error installing ingress-nginx: %v", err)
//	}
//	// sleep for few seconds - this is to make sure that the CRDs are properly applied
//	time.Sleep(20 * time.Second)
//
//	// Enabling Istio injection
//	spinner.SetNewAction("Enabling istio injection")
//	if err := kubernetes.ApplyLable("namespace", "default", "istio-injection=enabled",
//		true); err != nil {
//		return fmt.Errorf("error enabling istio injection: %v", err)
//	}
//
//	// Install istio
//	spinner.SetNewAction("Installing istio")
//	if err := InstallIstio(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
//		return fmt.Errorf("error installing istio: %v", err)
//	}
//
//	// Apply only knative serving CRD's
//	if err := ApplyKnativeCrds(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
//		return fmt.Errorf("error installing knative serving: %v", err)
//	}
//
//	// Apply controller CRDs
//	spinner.SetNewAction("Creating controller")
//	if err := InstallController(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts)); err != nil {
//		return fmt.Errorf("error creating cellery controller: %v", err)
//	}
//
//	spinner.SetNewAction("Configuring mysql")
//	if err := AddMysql(runtime.artifactsPath, runtime.isPersistentVolume); err != nil {
//		return fmt.Errorf("error configuring mysql: %v", err)
//	}
//
//	spinner.SetNewAction("Creating ConfigMaps")
//	if err := CreateGlobalGatewayConfigMaps(runtime.artifactsPath); err != nil {
//		return fmt.Errorf("error creating gateway configmaps: %v", err)
//	}
//	if err := CreateObservabilityConfigMaps(runtime.artifactsPath); err != nil {
//		return fmt.Errorf("error creating observability configmaps: %v", err)
//	}
//	if err := CreateIdpConfigMaps(runtime.artifactsPath); err != nil {
//		return fmt.Errorf("error creating idp configmaps: %v", err)
//	}
//
//	if runtime.isPersistentVolume {
//		spinner.SetNewAction("Creating Persistent Volume")
//		if err := createPersistentVolume(runtime.artifactsPath, runtime.hasNfsStorage); err != nil {
//			return fmt.Errorf("error creating persistent volume: %v", err)
//		}
//	}
//
//	if isCompleteSetup {
//		spinner.SetNewAction("Adding apim")
//		if err := addApim(runtime.artifactsPath, runtime.isPersistentVolume); err != nil {
//			return fmt.Errorf("error creating apim deployment: %v", err)
//		}
//		spinner.SetNewAction("Adding observability")
//		if err := addObservability(runtime.artifactsPath); err != nil {
//			return fmt.Errorf("error creating observability deployment: %v", err)
//		}
//	} else {
//		spinner.SetNewAction("Adding idp")
//		if err := addIdp(runtime.artifactsPath); err != nil {
//			return fmt.Errorf("error creating idp deployment: %v", err)
//		}
//	}
//	if !runtime.isLoadBalancerIngressMode {
//		if runtime.nodePortIpAddress != "" {
//			spinner.SetNewAction("Adding node port ip address")
//			originalIngressNginx, err := kubernetes.GetService("ingress-nginx", "ingress-nginx")
//			if err != nil {
//				return fmt.Errorf("error getting original ingress-nginx: %v", err)
//			}
//			updatedIngressNginx, err := kubernetes.GetService("ingress-nginx", "ingress-nginx")
//			if err != nil {
//				return fmt.Errorf("error getting updated ingress-nginx: %v", err)
//			}
//			updatedIngressNginx.Spec.ExternalIPs = append(updatedIngressNginx.Spec.ExternalIPs, runtime.nodePortIpAddress)
//
//			originalData, err := json.Marshal(originalIngressNginx)
//			if err != nil {
//				return fmt.Errorf("error marshalling original data: %v", err)
//			}
//			desiredData, err := json.Marshal(updatedIngressNginx)
//			if err != nil {
//				return fmt.Errorf("error marshalling desired data: %v", err)
//			}
//			patch, err := jsonpatch.CreatePatch(originalData, desiredData)
//			if err != nil {
//				return fmt.Errorf("error creating json patch: %v", err)
//			}
//			if len(patch) == 0 {
//				return fmt.Errorf("no changes in ingress-nginx to apply")
//			}
//			patchBytes, err := json.Marshal(patch)
//			if err != nil {
//				return fmt.Errorf("error marshalling json patch: %v", err)
//			}
//			kubernetes.JsonPatchWithNameSpace("svc", "ingress-nginx", string(patchBytes), "ingress-nginx")
//		}
//	}
//	spinner.Stop(true)
//	return nil
//}

func (runtime *CelleryRuntime) Create() error {
	spinner := util.StartNewSpinner("Creating cellery runtime")
    //Install istio crds and components.
	log.Printf("Deploying istio CRDs using istio-init chart")
	if err := util.ApplyHelmChartWithDefaultValues("istio-init", "istio-system"); err != nil {
		return fmt.Errorf("error installing istio crds: %v", err)
	}
	// sleep for few seconds - this is to make sure that the CRDs are properly applied
	time.Sleep(20 * time.Second)
	// Enabling Istio injection
	spinner.SetNewAction("Enabling istio injection")
	if err := kubernetes.ApplyLable("namespace", "default", "istio-injection=enabled",
		true); err != nil {
		return fmt.Errorf("error enabling istio injection: %v", err)
	}
	log.Printf("Deploying istio system using istio chart")
	if err := util.ApplyHelmChartWithDefaultValues("istio", "istio-system"); err != nil {
		return fmt.Errorf("error installing istio : %v", err)
	}
	//time.Sleep(20 * time.Second)

	log.Printf("Deploying knative system using knative-crd chart")
	if err := util.ApplyHelmChartWithDefaultValues("knative-crd", "default"); err != nil {
		return fmt.Errorf("error installing knative crds: %v", err)
	}

	log.Printf("Deploying ingress controller Nodeport system using ingress-controller chart")
	ingressControllerVals := IngressController{}
	ingVals, errVal := util.GetHelmChartDefaultValues("ingress-controller")
	if errVal != nil {
		log.Fatalf("error: %v", errVal)
	}
	errYaml := yaml.Unmarshal([]byte(ingVals), &ingressControllerVals)
	if errYaml != nil {
		log.Fatalf("error: %v", errYaml)
	}
	if !runtime.isLoadBalancerIngressMode {
		if runtime.nodePortIpAddress != "" {
			spinner.SetNewAction("Adding node port IP address")
			ingressControllerVals.NginxIngress.Controller.Service.Type = "NodePort"
			ingressControllerVals.NginxIngress.Controller.ExternalIPs = []string{runtime.nodePortIpAddress}
		}
	} else {
		ingressControllerVals.NginxIngress.Controller.Service.Type = "LoadBalancer"
	}
	controllerYamls, errcon := yaml.Marshal(&ingressControllerVals)
	if errcon != nil {
		log.Fatalf("error: %v", errcon)
	}
	if err := util.ApplyHelmChartWithCustomValues("ingress-controller", "ingress-nginx",
		"apply", string(controllerYamls)); err != nil {
		return fmt.Errorf("error installing ingress controller: %v", err)
	}

	log.Printf("Deploying cellery runtime using cellery-runtime chart")
	celleryValues := CelleryRuntimeVals{}
	chartName := "cellery-runtime"
	celleryVals, errCelVals := util.GetHelmChartDefaultValues(chartName)
	if errCelVals != nil {
		log.Fatalf("error: %v", errCelVals)
	}
	err := yaml.Unmarshal([]byte(celleryVals), &celleryValues)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	celleryValues.Mysql.Enabled = true
	if runtime.isPersistentVolume {
		celleryValues.Mysql.Persistence.Enabled = true
		if runtime.hasNfsStorage {
			celleryValues.Mysql.Nfs.Enabled = true
			celleryValues.Mysql.Nfs.ServerIp = runtime.nfs.NfsServerIp
			celleryValues.Mysql.Nfs.ShareLocation = runtime.nfs.FileShare
		}else {
			createFoldersRequiredForMysqlPvc()
			celleryValues.Mysql.LocalStorage.Enabled = true
		}
	} else {
		celleryValues.Mysql.Persistence.Enabled = false
	}

	spinner.SetNewAction("Creating controller")
	celleryValues.Controller.Enabled = true

	// Lable the node to support local persistence-volume
	if runtime.isPersistentVolume && !IsGcpRuntime() {
		nodeName, err := kubernetes.GetMasterNodeName()
		if err != nil {
			return fmt.Errorf("error getting master node name: %v", err)
		}
		if err := kubernetes.ApplyLable("nodes", nodeName, "disk=local", true); err != nil {
			return fmt.Errorf("error applying master node lable: %v", err)
		}
	}
	if !isCompleteSetup {
		celleryValues.Idp.Enabled = true
	} else {
		createFoldersRequiredForApimPvc()
		celleryValues.ApiManager.Enabled = true
		if runtime.isPersistentVolume {
			celleryValues.ApiManager.Persistence.Enabled = true
			if runtime.hasNfsStorage {
				celleryValues.ApiManager.Persistence.Media = "nfs"
				celleryValues.ApiManager.Persistence.NfsServerIp = runtime.nfs.NfsServerIp
				celleryValues.ApiManager.Persistence.SharedLocation = runtime.nfs.FileShare + "/" + "apim_repository_deployment_server"
			} else {
				celleryValues.ApiManager.Persistence.Media = "local-storage"
			}
		}
		celleryValues.Observability.Enabled = true
	}
	celleryYamls, errcon := yaml.Marshal(&celleryValues)
	if errcon != nil {
		log.Fatalf("error: %v", errcon)
	}
	//log.Printf(string(celleryYamls))
	if err := util.ApplyHelmChartWithCustomValues("cellery-runtime", "cellery-system", "apply", string(celleryYamls)); err != nil {
		return fmt.Errorf("error installing ingress controller: %v", err)
	}
	spinner.Stop(true)
	return nil
}

func UpdateRuntime(apiManagement, observability, knative, hpa Selection) error {
	spinner := util.StartNewSpinner("Updating cellery runtime")
	var err error
	observabilityEnabled, err := IsObservabilityEnabled()
	if err != nil {
		spinner.Stop(false)
		return err
	}
	if apiManagement != NoChange {
		// Remove observability if there was a change to apim
		if observabilityEnabled {
			err = DeleteComponent(Observability)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
		if apiManagement == Enable {
			err = DeleteComponent(IdentityProvider)
			if err != nil {
				spinner.Stop(false)
				return err
			}
			err = AddComponent(ApiManager)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		} else {
			err = DeleteComponent(ApiManager)
			if err != nil {
				spinner.Stop(false)
				return err
			}
			err = AddComponent(IdentityProvider)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
		// Add observability if there was a change to apim and there was already observability running before that
		if observabilityEnabled {
			err = AddComponent(Observability)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
	}
	if observability != NoChange {
		if observability == Enable {
			err = AddComponent(Observability)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		} else {
			err = DeleteComponent(Observability)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
	}
	if knative != NoChange {
		if knative == Enable {
			err = AddComponent(ScaleToZero)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		} else {
			err = DeleteComponent(ScaleToZero)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
	}
	if hpa != NoChange {
		if hpa == Enable {
			err = AddComponent(HPA)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		} else {
			err = DeleteComponent(HPA)
			if err != nil {
				spinner.Stop(false)
				return err
			}
		}
	}
	spinner.Stop(true)
	return nil
}

func AddComponent(component SystemComponent) error {
	switch component {
	case ApiManager:
		return addApim(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts), false)
	case IdentityProvider:
		return addIdp(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
	case Observability:
		return addObservability(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
	case ScaleToZero:
		return InstallKnativeServing(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
	case HPA:
		return InstallHPA(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
	default:
		return fmt.Errorf("unknown system componenet %q", component)
	}
}

//func DeleteComponent(component SystemComponent) error {
//	switch component {
//	case ApiManager:
//		return deleteApim(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
//	case IdentityProvider:
//		return deleteIdp(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
//	case Observability:
//		return deleteObservability(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
//	case ScaleToZero:
//		return deleteKnative()
//	case HPA:
//		return deleteHpa(filepath.Join(util.CelleryInstallationDir(), constants.K8sArtifacts))
//	default:
//		return fmt.Errorf("unknown system componenet %q", component)
//	}
//}

func DeleteComponent(component SystemComponent) error {
	switch component {
	case ApiManager:
		return deleteApim()
	case IdentityProvider:
		return deleteIdp()
	case Observability:
		return deleteObservability()
	case ScaleToZero:
		return deleteKnative()
	case HPA:
		return deleteHpa()
	default:
		return fmt.Errorf("unknown system componenet %q", component)
	}
}

func (runtime *CelleryRuntime) IsComponentEnabled(component SystemComponent) (bool, error) {
	switch component {
	case ApiManager:
		return IsApimEnabled()
	case Observability:
		return IsObservabilityEnabled()
	case ScaleToZero:
		return IsKnativeEnabled()
	case HPA:
		return IsHpaEnabled()
	default:
		return false, fmt.Errorf("unknown system componenet %q", component)
	}
}

func createFoldersRequiredForMysqlPvc() {
	// Backup folders
	util.RenameFile(filepath.Join(constants.RootDir, constants.VAR, constants.TMP, constants.CELLERY, constants.MySql),
		filepath.Join(constants.RootDir, constants.VAR, constants.TMP, constants.CELLERY, constants.MySql)+"-old")
	// Create folders required by the mysql PVC
	util.CreateDir(filepath.Join(constants.RootDir, constants.VAR, constants.TMP, constants.CELLERY, constants.MySql))
}

func createFoldersRequiredForApimPvc() {
	// Backup folders
	util.RenameFile(filepath.Join(constants.RootDir, constants.VAR, constants.TMP, constants.CELLERY,
		constants.ApimRepositoryDeploymentServer), filepath.Join(constants.RootDir, constants.VAR, constants.TMP,
		constants.CELLERY, constants.ApimRepositoryDeploymentServer)+"-old")
	// Create folders required by the APIM PVC
	util.CreateDir(filepath.Join(constants.RootDir, constants.VAR, constants.TMP, constants.CELLERY,
		constants.ApimRepositoryDeploymentServer))
}

func buildArtifactsPath(component SystemComponent, artifactsPath string) string {
	switch component {
	case ApiManager:
		return filepath.Join(artifactsPath, "global-apim")
	case IdentityProvider:
		return filepath.Join(artifactsPath, "global-idp")
	case Observability:
		return filepath.Join(artifactsPath, "observability")
	case Controller:
		return filepath.Join(artifactsPath, "controller")
	case System:
		return filepath.Join(artifactsPath, "system")
	case Mysql:
		return filepath.Join(artifactsPath, "mysql")
	case HPA:
		return filepath.Join(artifactsPath, "metrics-server/")
	default:
		return filepath.Join(artifactsPath)
	}
}

func IsGcpRuntime() bool {
	nodes, err := kubernetes.GetNodes()
	if err != nil {
		util.ExitWithErrorMessage("failed to check if runtime is gcp", err)
	}
	for _, node := range nodes.Items {
		version := node.Status.NodeInfo.KubeletVersion
		if strings.Contains(version, "gke") {
			return true
		}
	}
	return false
}

func WaitFor(checkKnative, hpaEnabled bool) {
	spinner := util.StartNewSpinner("Checking cluster status...")
	wtCluster, err := waitingTimeCluster()
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error getting waiting time for cluster", err)
	}
	err = kubernetes.WaitForCluster(wtCluster)
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error while checking cluster status", err)
	}
	spinner.SetNewAction("Cluster status...OK")
	spinner.Stop(true)

	spinner = util.StartNewSpinner("Checking runtime status (Istio)...")
	err = kubernetes.WaitForDeployments("istio-system", time.Minute*15)
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error while checking runtime status (Istio)", err)
	}
	spinner.SetNewAction("Runtime status (Istio)...OK")
	spinner.Stop(true)

	if checkKnative {
		spinner = util.StartNewSpinner("Checking runtime status (Knative Serving)...")
		err = kubernetes.WaitForDeployments("knative-serving", time.Minute*15)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Error while checking runtime status (Knative Serving)", err)
		}
		spinner.SetNewAction("Runtime status (Knative Serving)...OK")
		spinner.Stop(true)
	}

	if hpaEnabled {
		spinner = util.StartNewSpinner("Checking runtime status (Metrics server)...")
		err = kubernetes.WaitForDeployment("available", 900, "metrics-server", "kube-system")
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Error while checking runtime status (Metrics server)", err)
		}
		spinner.SetNewAction("Runtime status (Metrics server)...OK")
		spinner.Stop(true)
	}

	spinner = util.StartNewSpinner("Checking runtime status (Cellery)...")
	wrCellerySysterm, err := waitingTimeCellerySystem()
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error getting waiting time for cellery system", err)
	}
	err = kubernetes.WaitForDeployments("cellery-system", wrCellerySysterm)
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error while checking runtime status (Cellery)", err)
	}
	spinner.SetNewAction("Runtime status (Cellery)...OK")
	spinner.Stop(true)
}

func waitingTimeCluster() (time.Duration, error) {
	waitingTime := time.Minute * 60
	envVar := os.Getenv("CELLERY_CLUSTER_WAIT_TIME_MINUTES")
	if envVar != "" {
		wt, err := strconv.Atoi(envVar)
		if err != nil {
			return waitingTime, err
		}
		waitingTime = time.Duration(time.Minute * time.Duration(wt))
	}
	return waitingTime, nil
}

func waitingTimeCellerySystem() (time.Duration, error) {
	waitingTime := time.Minute * 15
	envVar := os.Getenv("CELLERY_SYSTEM_WAIT_TIME_MINUTES")
	if envVar != "" {
		wt, err := strconv.Atoi(envVar)
		if err != nil {
			return waitingTime, err
		}
		waitingTime = time.Duration(time.Minute * time.Duration(wt))
	}
	return waitingTime, nil
}
