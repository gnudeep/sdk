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

package main

import (
	"github.com/spf13/cobra"

	"github.com/cellery-io/sdk/components/cli/pkg/commands"
)

func newSetupCleanupExistingCommand() *cobra.Command {
	var confirmed = false
	var istio = false
	var knative = false
	var ingress = false
	var hpa = false
	cmd := &cobra.Command{
		Use:   "existing",
		Short: "Cleanup existing cluster setup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			commands.RunCleanupExisting(knative, istio, ingress, hpa, confirmed)
		},
		Example: "  cellery setup cleanup existing",
	}
	cmd.Flags().BoolVar(&istio, "istio", false, "Remove istio")
	cmd.Flags().BoolVar(&knative, "knative", false, "Remove knative-serving")
	cmd.Flags().BoolVar(&ingress, "ingress", false, "Remove ingress")
	cmd.Flags().BoolVar(&hpa, "hpa", false, "Remove hpa")
	cmd.Flags().BoolVarP(&confirmed, "assume-yes", "y", false, "Confirm setup creation")
	return cmd
}
