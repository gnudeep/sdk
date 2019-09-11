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

package kubectl

import (
	"os"
	"os/exec"

	"github.com/cellery-io/sdk/components/cli/pkg/constants"
	"github.com/cellery-io/sdk/components/cli/pkg/osexec"
)

func DeleteFileWithNamespace(file, namespace string) error {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"-f",
		file,
		"--ignore-not-found",
		"-n", namespace,
	)
	displayVerboseOutput(cmd)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteFile(file string) error {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"-f",
		file,
		"--ignore-not-found",
	)
	displayVerboseOutput(cmd)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteResource(kind, instance string) (string, error) {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		kind,
		instance,
		"--ignore-not-found",
	)
	displayVerboseOutput(cmd)
	return osexec.GetCommandOutput(cmd)
}

func DeleteNameSpace(nameSpace string) error {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"ns",
		nameSpace,
		"--ignore-not-found",
	)
	displayVerboseOutput(cmd)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteAllCells() error {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"cells",
		"--all",
		"--ignore-not-found",
	)
	displayVerboseOutput(cmd)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteCell(cellInstance string) (string, error) {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"cell",
		cellInstance,
	)
	displayVerboseOutput(cmd)
	return osexec.GetCommandOutput(cmd)
}

func DeletePersistedVolume(persistedVolume string) error {
	cmd := exec.Command(
		constants.KUBECTL,
		"delete",
		"pv",
		persistedVolume,
		"--ignore-not-found",
	)
	displayVerboseOutput(cmd)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
