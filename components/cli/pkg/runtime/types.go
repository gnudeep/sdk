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

type T struct {
	A string
	B struct {
		RenamedC int   `yaml:"c"`
		D        []int `yaml:",flow"`
	}
}

type CelleryRuntimeValues struct {
	Db struct {
		Carbon struct {
			UserName string `yaml:"wso2carbon"`
			UserPassword string `yaml:"wso2carbon"`
		}
	}


}