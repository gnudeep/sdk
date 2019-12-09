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

type SystemComponent string

const (
	ApiManager       SystemComponent = "ApiManager"
	IdentityProvider SystemComponent = "IdentityProvider"
	Observability    SystemComponent = "Observability"
	ScaleToZero      SystemComponent = "Scale to zero"
	HPA              SystemComponent = "Horizontal pod auto scalar"
	Controller       SystemComponent = "Controller"
	System           SystemComponent = "System"
	Mysql            SystemComponent = "Mysql"
)

type CelleryRuntimeVals struct {
	Global struct {
		CelleryRuntime struct {
			Db struct {
				CarbonDb struct {
					Username string `yaml:"username"`
					Password string `yaml:"password"`
				} `yaml:"carbon"`
			} `yaml:"db"`

			CarbonSystem struct {
				AdminUser struct {
					UserName     string `yaml:"username"`
					UserPassword string `yaml:"password"`
				} `yaml:"admin"`
			} `yaml:"carbon"`
		} `yaml:"celleryRuntime"`
	} `yaml:"global"`

	Mysql struct {
		Enabled      bool `yaml:"enabled"`
		RootPassword string `yaml:"rootPassword"`

		Persistence struct {
			Enabled      bool `yaml:"enabled"`
			StorageClass string `yaml:"storageClass"`
			AccessMode   string `yaml:"accessMode"`
			Size         string `yaml:"size"`
			SubPath      string `yaml:"subPath"`
		} `yaml:"persistence"`

		Nfs struct {
			Enabled        bool `yaml:"enabled"`
			ServerIp       string `yaml:"serverIp"`
			ShareLocation string `yaml:"shareLocation"`
		} `yaml:"nfs"`

		LocalStorage struct {
			Enabled     bool `yaml:"enabled"`
			StoragePath string `yaml:"storagePath"`
		} `yaml:"localStorage"`
	} `yaml:"mysql"`

	Controller struct {
		Enabled      bool `yaml:"enabled"`
	} `yaml:"controller"`

	ApiManager struct {
		Enabled      bool `yaml:"enabled"`
		Persistence struct {
			Enabled bool `yaml:"enabled"`
			Media string `yaml:"media"`
			NfsServerIp   string `yaml:"nfsServerIp"`
			SharedLocation	string `yaml:"sharedLocation"`
		} `yaml:"persistence"`
	} `yaml:"apim"`

	Idp struct {
		Enabled      bool `yaml:"enabled"`
	} `yaml:"idp"`

	Observability struct {
		Enabled      bool `yaml:"enabled"`
	} `yaml:"observability-portal"`

	SpWorker struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"sp-worker"`

	Grafana struct {
		Enabled      bool `yaml:"enabled"`
	} `yaml:"grafana"`

	Prometheus struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"prometheus"`

	MixerAdapter struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"mixer-adapter"`
}

type IngressController struct {
	NginxIngress struct {
		Enabled    bool `yaml:"enabled"`
		Controller struct {
			ImageName struct {
				Tag string `yaml:"tag"`
			} `yaml:"image"`
			Service struct {
				Type string `yaml:"type"`
			} `yaml:"service"`
			ExternalIPs []string `yaml:"externalIPs"`
		} `yaml:"controller"`
	} `yaml:"nginx-ingress"`
}