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
	"os"
	"path/filepath"
	"regexp"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/oxequa/interact"

	"cellery.io/cellery/components/cli/cli"
	"cellery.io/cellery/components/cli/pkg/constants"
	"cellery.io/cellery/components/cli/pkg/runtime"
	"cellery.io/cellery/components/cli/pkg/util"
)

func createOnExistingCluster(cli cli.Cli) error {
	var isPersistentVolume = false
	var hasNfsStorage = false
	var isLoadBalancerIngressMode = false
	var isBackSelected = false
	var nfs runtime.Nfs
	var db runtime.MysqlDb
	var nodePortIpAddress = ""
	cellTemplate := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U000027A4 {{ .| bold }}",
		Inactive: "  {{ . | faint }}",
		Help:     util.Faint("[Use arrow keys]"),
	}

	cellPrompt := promptui.Select{
		Label:     util.YellowBold("?") + " Select the type of runtime",
		Items:     []string{constants.PersistentVolume, constants.NonPersistentVolume, setupBack},
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
	if value == constants.PersistentVolume {
		isPersistentVolume = true
		hasNfsStorage, isBackSelected, err = util.GetYesOrNoFromUser(fmt.Sprintf("Use NFS server"),
			true)
		if err != nil {
			util.ExitWithErrorMessage("Failed to select an option", err)
		}
		if hasNfsStorage {
			nfs, db, err = getPersistentVolumeDataWithNfs()
		}
		if isBackSelected {
			createOnExistingCluster(cli)
			return nil
		}
	}
	isCompleteSetup, isBackSelected := util.IsCompleteSetupSelected()
	runtime.SetCompleteSetup(isCompleteSetup)
	if isBackSelected {
		createOnExistingCluster(cli)
		return nil
	}
	isLoadBalancerIngressMode, isBackSelected = util.IsLoadBalancerIngressTypeSelected()
	if isBackSelected {
		createOnExistingCluster(cli)
		return nil
	}
	if !isLoadBalancerIngressMode {
		nodePortIpAddress = getNodePortIpAddress()
		isNodePortIpAddressValid, err := regexp.MatchString(fmt.Sprintf("^%s$|^$", constants.IpAddressPattern),
			nodePortIpAddress)
		if err != nil || !isNodePortIpAddressValid {
			util.ExitWithErrorMessage("Error creating cellery runtime", fmt.Errorf("expects a valid "+
				"nodeport ip address, received %s", nodePortIpAddress))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get user input: %v", err)
	}
	return RunSetupCreateOnExistingCluster(cli, isPersistentVolume, hasNfsStorage, isLoadBalancerIngressMode, nfs, db, nodePortIpAddress)
}

func RunSetupCreateOnExistingCluster(cli cli.Cli, isPersistentVolume, hasNfsStorage, isLoadBalancerIngressMode bool,
	nfs runtime.Nfs, db runtime.MysqlDb, nodePortIpAddress string) error {
	artifactsPath := filepath.Join(cli.FileSystem().UserHome(), constants.CelleryHome, constants.K8sArtifacts)
	os.RemoveAll(artifactsPath)
	util.CopyDir(filepath.Join(cli.FileSystem().CelleryInstallationDir(), constants.K8sArtifacts), artifactsPath)
	helmCharPath := filepath.Join(cli.FileSystem().UserHome(), constants.CelleryHome, constants.HelmCarts)
	os.RemoveAll(helmCharPath)
	util.CopyDir(filepath.Join(filepath.Join(cli.FileSystem().CelleryInstallationDir()), constants.HelmCarts), helmCharPath)

	cli.Runtime().SetArtifactsPath(artifactsPath)
	cli.Runtime().SetPersistentVolume(isPersistentVolume)
	cli.Runtime().SetHasNfsStorage(hasNfsStorage)
	cli.Runtime().SetLoadBalancerIngressMode(isLoadBalancerIngressMode)
	cli.Runtime().SetNfs(nfs)
	cli.Runtime().SetDb(db)
	cli.Runtime().SetNodePortIpAddress(nodePortIpAddress)

	if err := cli.Runtime().Create(); err != nil {
		return fmt.Errorf("failed to deploy cellery runtime, %v", err)
	}
	runtime.WaitFor(false, false)
	return nil
}

func getPersistentVolumeDataWithNfs() (runtime.Nfs, runtime.MysqlDb, error) {
	prefix := util.CyanBold("?")
	nfsServerIp := ""
	fileShare := ""
	dbHostName := ""
	dbUserName := ""
	dbPassword := ""
	err := interact.Run(&interact.Interact{
		Before: func(c interact.Context) error {
			c.SetPrfx(color.Output, prefix)
			return nil
		},
		Questions: []*interact.Question{
			{
				Before: func(c interact.Context) error {
					c.SetPrfx(nil, util.CyanBold("?"))
					return nil
				},
				Quest: interact.Quest{
					Msg: util.Bold("NFS server IP: "),
				},
				Action: func(c interact.Context) interface{} {
					nfsServerIp, _ = c.Ans().String()
					return nil
				},
			},
			{
				Before: func(c interact.Context) error {
					c.SetPrfx(nil, util.CyanBold("?"))
					return nil
				},
				Quest: interact.Quest{
					Msg: util.Bold("File share name: "),
				},
				Action: func(c interact.Context) interface{} {
					fileShare, _ = c.Ans().String()
					return nil
				},
			},
			{
				Before: func(c interact.Context) error {
					c.SetPrfx(nil, util.CyanBold("?"))
					return nil
				},
				Quest: interact.Quest{
					Msg: util.Bold("Database host: "),
				},
				Action: func(c interact.Context) interface{} {
					dbHostName, _ = c.Ans().String()
					return nil
				},
			},
		},
	})
	if err != nil {
		util.ExitWithErrorMessage("Error occurred while getting user input", err)
	}
	dbUserName, dbPassword, err = util.RequestCredentials("Mysql", "")
	if err != nil {
		util.ExitWithErrorMessage("Error occurred while getting user input", err)
	}
	return runtime.Nfs{NfsServerIp: nfsServerIp, FileShare: fileShare},
		runtime.MysqlDb{DbHostName: dbHostName, DbUserName: dbUserName, DbPassword: dbPassword}, nil
}

func getNodePortIpAddress() string {
	prefix := util.CyanBold("?")
	nodePortIpAddress := ""
	err := interact.Run(&interact.Interact{
		Before: func(c interact.Context) error {
			c.SetPrfx(color.Output, prefix)
			return nil
		},
		Questions: []*interact.Question{
			{
				Before: func(c interact.Context) error {
					c.SetDef("", util.Faint("[Press enter to use default NodePort ip address]"))
					c.SetPrfx(nil, util.CyanBold("?"))
					return nil
				},
				Quest: interact.Quest{
					Msg: util.Bold("NodePort Ip address: "),
				},
				Action: func(c interact.Context) interface{} {
					nodePortIpAddress, _ = c.Ans().String()
					return nil
				},
			},
		},
	})
	if err != nil {
		util.ExitWithErrorMessage("Error occurred while getting nodePort id address from user", err)
	}
	return nodePortIpAddress
}
