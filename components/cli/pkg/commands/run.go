/*
 * Copyright (c) 2018 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
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

package commands

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/cellery-io/sdk/components/cli/pkg/constants"
	"github.com/cellery-io/sdk/components/cli/pkg/image"
	"github.com/cellery-io/sdk/components/cli/pkg/kubectl"
	celleryRuntime "github.com/cellery-io/sdk/components/cli/pkg/runtime"
	"github.com/cellery-io/sdk/components/cli/pkg/util"
	"github.com/cellery-io/sdk/components/cli/pkg/version"
)

// RunRun starts Cell instance (along with dependency instances if specified by the user)
// This also support linking instances to parts of the dependency tree
// This command also strictly validates whether the requested Cell (and the dependencies are valid)
func RunRun(cellImageTag string, instanceName string, startDependencies bool, shareDependencies bool,
	dependencyLinks []string, envVars []string, assumeYes bool) {
	spinner := util.StartNewSpinner("Extracting Cell Image " + util.Bold(cellImageTag))
	parsedCellImage, err := image.ParseImageTag(cellImageTag)
	if err != nil {
		util.ExitWithErrorMessage("Error occurred while parsing cell image", err)
	}
	imageDir, err := ExtractImage(parsedCellImage, true, spinner)
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error occurred while extracting image", err)
	}
	defer func() {
		err = os.RemoveAll(imageDir)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Error occurred while cleaning up", err)
		}
	}()

	// Reading Cell Image metadata
	spinner.SetNewAction("Reading Cell Image " + util.Bold(cellImageTag))
	metadataFileContent, err := ioutil.ReadFile(filepath.Join(imageDir, constants.ZIP_ARTIFACTS, "cellery",
		"metadata.json"))
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error occurred while reading Cell Image metadata", err)
	}
	cellImageMetadata := &image.MetaData{}
	err = json.Unmarshal(metadataFileContent, cellImageMetadata)
	if err != nil {
		spinner.Stop(false)
		util.ExitWithErrorMessage("Error occurred while reading Cell Image metadata", err)
	}

	if instanceName == "" {
		// Setting a unique instance name since it is not provided
		instanceName, err = generateRandomInstanceName(cellImageMetadata)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Error occurred while preparing", err)
		}
	} else {
		_, err := kubectl.GetCell(instanceName)
		if err == nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage(fmt.Sprintf("Instance %s already exists", instanceName),
				fmt.Errorf("instance to be created should not be present in the runtime, "+
					"instance %s is already available in the runtime", instanceName))
		} else if !strings.Contains(err.Error(), "not found") {
			util.ExitWithErrorMessage(fmt.Sprintf("Error occurred while checking whether instance %s "+
				"exists in the runtime", instanceName), err)
		}
	}
	fmt.Printf("\r\x1b[2K\n%s: %s\n\n", util.Bold("Main Instance"), instanceName)

	var parsedDependencyLinks []*dependencyAliasLink
	if len(dependencyLinks) > 0 {
		// Parsing the dependency links list
		spinner.SetNewAction("Validating dependency links")
		for _, link := range dependencyLinks {
			var dependencyLink *dependencyAliasLink
			linkSplit := strings.Split(link, ":")
			if strings.Contains(linkSplit[0], ".") {
				instanceSplit := strings.Split(linkSplit[0], ".")
				dependencyLink = &dependencyAliasLink{
					Instance:           instanceSplit[0],
					DependencyAlias:    instanceSplit[1],
					DependencyInstance: linkSplit[1],
				}
			} else {
				dependencyLink = &dependencyAliasLink{
					DependencyAlias:    linkSplit[0],
					DependencyInstance: linkSplit[1],
				}
			}
			cellInstance, err := kubectl.GetCell(dependencyLink.DependencyInstance)
			if err != nil && !strings.Contains(err.Error(), "not found") {
				spinner.Stop(false)
				util.ExitWithErrorMessage("Error occurred while validating dependency links", err)
			}
			dependencyLink.IsRunning = err == nil && cellInstance.CellStatus.Status == "Ready"
			parsedDependencyLinks = append(parsedDependencyLinks, dependencyLink)
		}
		err = validateDependencyLinks(instanceName, cellImageMetadata, parsedDependencyLinks)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Invalid dependency links", err)
		}
	}

	instanceEnvVars := map[string][]*environmentVariable{}
	if len(envVars) > 0 {
		// Parsing environment variables
		spinner.SetNewAction("Validating environment variables")
		for _, envVar := range envVars {
			var targetInstance string
			var envVarKey string
			var envVarValue string

			// Parsing the environment variable
			r := regexp.MustCompile(fmt.Sprintf("^%s$", constants.CLI_ARG_ENV_VAR_PATTERN))
			matches := r.FindStringSubmatch(envVar)
			if matches != nil {
				for i, name := range r.SubexpNames() {
					if i != 0 && name != "" && matches[i] != "" { // Ignore the whole regexp match and unnamed groups
						switch name {
						case "instance":
							targetInstance = matches[i]
						case "key":
							envVarKey = matches[i]
						case "value":
							envVarValue = matches[i]
						}
					}
				}
			}
			if targetInstance == "" {
				targetInstance = instanceName
			}
			parsedEnvVar := &environmentVariable{
				Key:   envVarKey,
				Value: envVarValue,
			}

			// Validating whether the instance of the environment var is provided as an instance of a link
			if targetInstance != instanceName {
				isInstanceProvided := false
				isInstanceToBeStarted := false
				for _, link := range parsedDependencyLinks {
					if targetInstance == link.DependencyInstance {
						isInstanceProvided = true
						isInstanceToBeStarted = !link.IsRunning
						break
					}
				}
				if !isInstanceProvided {
					spinner.Stop(false)
					util.ExitWithErrorMessage("Invalid environment variable",
						fmt.Errorf("the instance of the environment variables should be provided as a "+
							"dependency link, instance %s of the environment variable %s not found", targetInstance,
							parsedEnvVar.Key))
				} else if !isInstanceToBeStarted {
					spinner.Stop(false)
					util.ExitWithErrorMessage("Invalid environment variable",
						fmt.Errorf("the instance of the environment should be an instance to be "+
							"created, instance %s is already available in the runtime", targetInstance))
				}
			}

			if _, hasKey := instanceEnvVars[targetInstance]; !hasKey {
				instanceEnvVars[targetInstance] = []*environmentVariable{}
			}
			instanceEnvVars[targetInstance] = append(instanceEnvVars[targetInstance], parsedEnvVar)
		}
	}

	var mainNode *dependencyTreeNode
	if startDependencies {
		spinner.SetNewAction("Generating dependency tree")
		dependencyTree, err := generateDependencyTree(instanceName, cellImageMetadata, parsedDependencyLinks,
			shareDependencies)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Error occurred while generating the dependency tree", err)
		}
		spinner.SetNewAction("Validating dependency tree")
		err = validateDependencyTreeLinks(dependencyTree)
		if err != nil {
			util.ExitWithErrorMessage("Invalid instance linking", err)
		}
		spinner.SetNewAction("Validating requirements")
		err = validateDependencyTreeRequirements(dependencyTree)
		if err != nil {
			util.ExitWithErrorMessage("Requirements unmet", err)
		}
		spinner.SetNewAction("")
		err = confirmDependencyTree(dependencyTree, assumeYes)
		if err != nil {
			spinner.Stop(false)
			util.ExitWithErrorMessage("Failed to confirm the dependency tree", err)
		}
		spinner.SetNewAction("Starting dependencies")
		startDependencyTree(parsedCellImage.Registry, dependencyTree, spinner, instanceEnvVars)
		if err != nil {
			util.ExitWithErrorMessage("Failed to start dependencies", err)
		}
		mainNode = dependencyTree
	} else {
		spinner.SetNewAction("Validating dependencies")
		immediateDependencies := map[string]*dependencyTreeNode{}
		// Check if the provided links are immediate dependencies of the root Cell
		for _, link := range parsedDependencyLinks {
			if !link.IsRunning {
				// When running without dependencies all the linked instances should be running in the runtime
				// Therefore the provided links are invalid
				spinner.Stop(false)
				util.ExitWithErrorMessage("Invalid link",
					fmt.Errorf("all the instances should be avaialable in the runtime when running "+
						"without depedencies, instance %s not available in the runtime", link.DependencyInstance))
			} else if link.Instance == "" || link.Instance == instanceName {
				var dependencyMetadata *image.MetaData
			cellImageMetadataLoop:
				for _, componentMetadata := range cellImageMetadata.Components {
					for alias, metadata := range componentMetadata.Dependencies.Cells {
						if link.DependencyAlias == alias {
							dependencyMetadata = metadata
							break cellImageMetadataLoop
						}
					}
				}
				if dependencyMetadata != nil {
					immediateDependencies[link.DependencyAlias] = &dependencyTreeNode{
						Instance:  link.DependencyInstance,
						MetaData:  dependencyMetadata,
						IsShared:  false,
						IsRunning: link.IsRunning,
					}
				} else {
					// If cellImageMetadata does not contain the provided link, there is a high chance that the user
					// made a mistake in the command. Therefore, this is validated strictly
					var allowedAliases []string
					for _, componentMetadata := range cellImageMetadata.Components {
						for alias := range componentMetadata.Dependencies.Cells {
							isAlreadyPresent := false
							for _, addedAlias := range allowedAliases {
								if addedAlias == alias {
									isAlreadyPresent = true
									break
								}
							}
							if !isAlreadyPresent {
								allowedAliases = append(allowedAliases, alias)
							}
						}
					}
					spinner.Stop(false)
					util.ExitWithErrorMessage("Invalid links",
						fmt.Errorf("only aliases of the main Cell instance %s: [%s] are allowed when running "+
							"without starting dependencies, received %s", instanceName,
							strings.Join(allowedAliases, ", "), link.DependencyAlias))
				}
			} else {
				// If the instance of the link (<instance>.<alias>:<dependency>), it should match the main instance
				// because the user had not instructed to start the whole dependency tree
				spinner.Stop(false)
				util.ExitWithErrorMessage("Invalid links",
					fmt.Errorf("only the main Cell instance %s is allowed when running "+
						"without starting dependencies, received unknown instance %s", instanceName, link.Instance))
			}
		}

		// Check if instances are provided for all the dependencies of the root Cell
		for _, componentMetadata := range cellImageMetadata.Components {
			for alias := range componentMetadata.Dependencies.Cells {
				isLinkProvided := false
				for _, link := range parsedDependencyLinks {
					if link.DependencyAlias == alias {
						isLinkProvided = true
						break
					}
				}
				if !isLinkProvided {
					// If a link is not provided for a particular dependency, the main instance cannot be started.
					// The links is required for the main instance to discover the dependency in the runtime
					spinner.Stop(false)
					util.ExitWithErrorMessage("Links for all the dependencies not found",
						fmt.Errorf("required link for alias %s in instance %s not found", alias, instanceName))
				}
			}
		}
		mainNode = &dependencyTreeNode{
			Instance:     instanceName,
			MetaData:     cellImageMetadata,
			IsRunning:    false,
			IsShared:     false,
			Dependencies: immediateDependencies,
		}
		spinner.SetNewAction("Validating links")
		err = validateDependencyTreeLinks(mainNode)
		if err != nil {
			util.ExitWithErrorMessage("Invalid instance linking", err)
		}
		spinner.SetNewAction("Validating requirements")
		err = validateDependencyTreeRequirements(mainNode)
		if err != nil {
			util.ExitWithErrorMessage("Requirements unmet", err)
		}
		spinner.SetNewAction("")
		err = confirmDependencyTree(mainNode, assumeYes)
		if err != nil {
			util.ExitWithErrorMessage("Failed to confirm the dependency tree", err)
		}
	}

	spinner.SetNewAction("Starting main instance " + util.Bold(instanceName))
	err = startCellInstance(imageDir, instanceName, mainNode, instanceEnvVars[instanceName])
	if err != nil {
		util.ExitWithErrorMessage("Failed to start Cell instance "+instanceName, err)
	}

	spinner.Stop(true)
	util.PrintSuccessMessage(fmt.Sprintf("Successfully deployed cell image: %s", util.Bold(cellImageTag)))
	util.PrintWhatsNextMessage("list running cells", "cellery list instances")
}

// validateDependencyLinks validates the dependency tree of the root instance
func validateDependencyLinks(rootInstance string, rootMetaData *image.MetaData,
	dependencyLinks []*dependencyAliasLink) error {
	// Validating the links provided by the user
	for _, link := range dependencyLinks {
		if link.Instance == "" {
			// This checks for duplicate aliases without parent instance and whether their Cell Images match.
			// If the duplicate aliases have matching Cell Images, then they can share the instance.
			// However, if duplicate aliases are present without parent instances and referring to different
			// Cell Images, the links should be more specific using parent instance
			var validateSubtree func(metadata *image.MetaData) error
			// cellImage is used to store the Cell image which is referred to by this link. This is used to validate
			// whether the Cell images of the duplicated aliases (without the parent instance) match.
			var cellImage *image.CellImage
			// This is used to store the instances which were shared due to the user providing a non specific link
			// (without the parent instance) which is duplicated in the dependency tree. A warning is shown later to
			// the user about these instances since this could be a mistake
			userSpecifiedSharedInstances := map[string]string{}
			validateSubtree = func(metadata *image.MetaData) error {
				for _, componentMetadata := range metadata.Components {
					for alias, dependencyMetadata := range componentMetadata.Dependencies.Cells {
						if alias == link.DependencyAlias {
							if cellImage == nil {
								// This is the first time the alias was found in the dependency tree.
								// (Since the Cell Image was not set)
								cellImage = &image.CellImage{
									Organization: dependencyMetadata.Organization,
									ImageName:    dependencyMetadata.Name,
									ImageVersion: dependencyMetadata.Version,
								}
							} else {
								// Check if the provided alias which is duplicated in the dependency
								// tree is the same image
								if cellImage.Organization != dependencyMetadata.Organization ||
									cellImage.ImageName != dependencyMetadata.Name ||
									cellImage.ImageVersion != dependencyMetadata.Version {
									return fmt.Errorf("duplicated alias %s in dependency tree, referes to "+
										"different images %s/%s:%s and %s/%s:%s, provided aliases which are "+
										"duplicated in the depedencies should be more specific using parent "+
										"instance", alias, cellImage.Organization, cellImage.ImageName,
										cellImage.ImageVersion, dependencyMetadata.Organization,
										dependencyMetadata.Name, dependencyMetadata.Version)
								} else {
									// Since the Cell Image is the same in both aliases the instance will be reused
									// The instance is stored to show a warning to the user later
									if _, hasKey := userSpecifiedSharedInstances[link.DependencyAlias]; !hasKey {
										userSpecifiedSharedInstances[link.DependencyAlias] = link.DependencyInstance
									}
								}
							}
						}
						err := validateSubtree(dependencyMetadata)
						if err != nil {
							return err
						}
					}
				}
				return nil
			}
			err := validateSubtree(rootMetaData)
			if err != nil {
				return err
			}
			for alias, instance := range userSpecifiedSharedInstances {
				// Warning the user about shared instances due to duplicated aliases
				fmt.Printf("\r\x1b[2K%s Using a shared instance %s for duplicated alias %s\n",
					util.YellowBold("\U000026A0"), util.Bold(instance), util.Bold(alias))
			}
		} else {
			// If the link has a parent instance, this checks if the parent instance had been provided by the user
			// All used parent instances should be specified beforehand as the instance of another alias
			var isLinkParentInstanceProvided bool
			if rootInstance == link.Instance {
				isLinkParentInstanceProvided = true
			} else {
				// Checking if the parent instance in the link is provided as an instance of another alias
				for _, userSpecifiedLink := range dependencyLinks {
					if link.Instance == userSpecifiedLink.DependencyInstance {
						isLinkParentInstanceProvided = true
						break
					}
				}
			}
			if !isLinkParentInstanceProvided {
				// The user is referring to an instance which is not provided as a link which could be a bug
				return fmt.Errorf("all parent instances of the provided links should be explicitly given "+
					"as an instance of another alias, instance %s not provided", link.Instance)
			}
		}
	}
	return nil
}

// generateDependencyOrder reads the metadata and generates a proper start up order for dependencies
func generateDependencyTree(rootInstance string, rootMetaData *image.MetaData, dependencyLinks []*dependencyAliasLink,
	shareDependencies bool) (*dependencyTreeNode, error) {
	// aliasToTreeNodeMap is used to keep track of the already created user provided tree nodes.
	// The key of the is the alias and the value is the tree node.
	// The alias is prefixed by the instance only if the user specified the parent instance as well.
	aliasToTreeNodeMap := map[string]*dependencyTreeNode{}

	// generatedInstanceTreeNodes are used to keep track of the instances automatically generated
	// These will be shared among the auto generated instances based on "shareDependencies" environment variable
	var generatedInstanceTreeNodes []*dependencyTreeNode

	// This is used to keep track of the used links. If an unused link is provided, this could be a mistake made by the
	// user. Therefore this is validated.
	var usedDependencyLinks []*dependencyAliasLink

	// traverseDependencies traverses through the dependency tree and populates the startup order considering the
	// relationship between dependencies
	var traverseDependencies func(instance string, metaData *image.MetaData, treeNode *dependencyTreeNode) error
	traverseDependencies = func(instance string, metaData *image.MetaData, treeNode *dependencyTreeNode) error {
		for _, componentMetaData := range metaData.Components {
			for alias, dependencyMetaData := range componentMetaData.Dependencies.Cells {
				var dependencyNode *dependencyTreeNode

				// Check if the dependency link is provided by the user
				for _, link := range dependencyLinks {
					if alias == link.DependencyAlias && (link.Instance == "" || link.Instance == instance) {
						var aliasPrefix string
						if link.Instance != "" {
							aliasPrefix = link.Instance + "."
						}
						key := aliasPrefix + alias

						if node, hasKey := aliasToTreeNodeMap[key]; hasKey {
							// Since the alias is already present in the map, the instance will be shared
							dependencyNode = node
							dependencyNode.IsShared = true
						} else {
							dependencyNode = &dependencyTreeNode{
								Instance:     link.DependencyInstance,
								MetaData:     dependencyMetaData,
								Dependencies: map[string]*dependencyTreeNode{},
								IsShared:     false,
								IsRunning:    link.IsRunning,
							}
							aliasToTreeNodeMap[key] = dependencyNode
							usedDependencyLinks = append(usedDependencyLinks, link)
						}
						break
					}
				}

				if dependencyNode == nil {
					if shareDependencies {
						// Check if an instance had been already allocated for this image
						for _, existingNode := range generatedInstanceTreeNodes {
							if existingNode.MetaData.Organization == dependencyMetaData.Organization &&
								existingNode.MetaData.Name == dependencyMetaData.Name &&
								existingNode.MetaData.Version == dependencyMetaData.Version {
								dependencyNode = existingNode
								existingNode.IsShared = true
							}
						}
					}

					if dependencyNode == nil {
						// Since no suitable instance that can be used is present, a random name is generated
						dependencyInstance, err := generateRandomInstanceName(dependencyMetaData)
						if err != nil {
							return err
						}
						dependencyNode = &dependencyTreeNode{
							Instance:     dependencyInstance,
							MetaData:     dependencyMetaData,
							Dependencies: map[string]*dependencyTreeNode{},
							IsShared:     false,
							IsRunning:    false,
						}
						generatedInstanceTreeNodes = append(generatedInstanceTreeNodes, dependencyNode)
					}
				}

				// Traversing the dependencies (Depth First Search for start up items)
				treeNode.Dependencies[alias] = dependencyNode
				err := traverseDependencies(dependencyNode.Instance, dependencyMetaData, dependencyNode)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	dependencyTreeRoot := &dependencyTreeNode{
		Instance:     rootInstance,
		MetaData:     rootMetaData,
		Dependencies: map[string]*dependencyTreeNode{},
		IsShared:     false,
		IsRunning:    false,
	}
	err := traverseDependencies(rootInstance, rootMetaData, dependencyTreeRoot)
	if err != nil {
		return nil, err
	}

	// Failing if unused dependency links are present. Unused dependency links are an indication of a user mistake
	for _, link := range dependencyLinks {
		isLinkUsed := false
		for _, usedLink := range usedDependencyLinks {
			if link.Instance == usedLink.Instance && link.DependencyAlias == usedLink.DependencyAlias &&
				link.DependencyInstance == usedLink.DependencyInstance {
				isLinkUsed = true
			}
		}
		if !isLinkUsed {
			// Unused links is a possible mistake done by the user. Therefore this is validated.
			var linkString string
			if link.Instance != "" {
				linkString += link.Instance + "."
			}
			linkString += fmt.Sprintf("%s:%s", link.DependencyAlias, link.DependencyInstance)
			return nil, fmt.Errorf("unused links should not be provided, link %s is not used", linkString)
		}
	}
	return dependencyTreeRoot, nil
}

// validateDependencyTreeLinks validates a generated dependency tree's links
func validateDependencyTreeLinks(treeRoot *dependencyTreeNode) error {
	// This is used to store the instances provided by the user and later validate with the runtime and check
	// whether the Cell Image of the instance matches with the linking provided by the user.
	instanceToNodeMap := map[string]*dependencyTreeNode{}

	// Validate whether the Cell Image of all the specified instances match
	var validateDependencySubtreeOffline func(subTreeRoot *dependencyTreeNode) error
	validateDependencySubtreeOffline = func(subTreeRoot *dependencyTreeNode) error {
		for _, dependency := range subTreeRoot.Dependencies {
			if node, hasKey := instanceToNodeMap[dependency.Instance]; hasKey {
				if node.MetaData.Organization != dependency.MetaData.Organization ||
					node.MetaData.Name != dependency.MetaData.Name ||
					node.MetaData.Version != dependency.MetaData.Version {
					// The user had pointed using links to share an instance with different Cell Images
					return fmt.Errorf("instance %s cannot be shared by different Cell Images %s/%s:%s and %s/%s:%s",
						dependency.Instance,
						dependency.MetaData.Organization, dependency.MetaData.Name, dependency.MetaData.Version,
						node.MetaData.Organization, node.MetaData.Name, node.MetaData.Version)
				}
			} else {
				instanceToNodeMap[dependency.Instance] = dependency
			}
			err := validateDependencySubtreeOffline(dependency)
			if err != nil {
				return err
			}
		}
		return nil
	}
	instanceToNodeMap[treeRoot.Instance] = treeRoot
	err := validateDependencySubtreeOffline(treeRoot)
	if err != nil {
		return err
	}

	// Validating whether the instances running in the runtime match the linked image
	for instance, node := range instanceToNodeMap {
		if node.IsRunning {
			cellInstance, err := kubectl.GetCell(instance)
			if err == nil && cellInstance.CellStatus.Status == "Ready" {
				if cellInstance.CellMetaData.Annotations.Organization != node.MetaData.Organization ||
					cellInstance.CellMetaData.Annotations.Name != node.MetaData.Name ||
					cellInstance.CellMetaData.Annotations.Version != node.MetaData.Version {
					// If the instance in the runtime and the user link for the instance refers to different images,
					// the linking is invalid.
					return fmt.Errorf("provided instance %s is required to be of type %s/%s:%s, "+
						"instance available in the runtime is from cell image %s/%s:%s",
						instance, node.MetaData.Organization, node.MetaData.Name, node.MetaData.Version,
						cellInstance.CellMetaData.Annotations.Organization, cellInstance.CellMetaData.Annotations.Name,
						cellInstance.CellMetaData.Annotations.Version)
				}
			} else if err != nil && !strings.Contains(err.Error(), "not found") {
				// If an error occurred which does not include not found (eg:- connection refused, insufficient
				// permissions), the run with dependencies task should fail
				return fmt.Errorf("failed to check whether instance %s exists in the runtime due to %v",
					instance, err)
			} else {
				return fmt.Errorf("instance %s is not available in the runtime", instance)
			}
		}
	}
	return nil
}

// validateDependencyTreeRequirements validates whether the requirements of all the cells to be
// started are met by the runtime.
func validateDependencyTreeRequirements(treeRoot *dependencyTreeNode) error {
	var validatedImages []string

	// Validate whether the runtime satisfies all the requirements of the subtree
	var validateDependencySubtreeOffline func(subTreeRoot *dependencyTreeNode) error
	validateDependencySubtreeOffline = func(subTreeRoot *dependencyTreeNode) error {
		alreadyValidated := false
		imageToBeValidated := fmt.Sprintf("%s/%s:%s", subTreeRoot.MetaData.Organization, subTreeRoot.MetaData.Name,
			subTreeRoot.MetaData.Version)
		for _, validatedImage := range validatedImages {
			if validatedImage == imageToBeValidated {
				alreadyValidated = true
				break
			}
		}

		if !alreadyValidated {
			if subTreeRoot.MetaData.ZeroScalingRequired {
				// Check K-Native is installed.
				if ok, err := celleryRuntime.IsKnativeEnabled(); err == nil {
					if !ok {
						return fmt.Errorf("cell contains zero-scaling components, " +
							"but zero-scaling is not enabled in Cellery runtime")
					}
				} else {
					util.ExitWithErrorMessage("Error occurred while checking whether zero-scaling is enabled.", err)
				}
			}
			if subTreeRoot.MetaData.AutoScalingRequired {
				// Check metrics-server is installed.
				if ok, err := celleryRuntime.IsHpaEnabled(); err == nil {
					if !ok {
						util.PrintWarningMessage(fmt.Sprintf("Cell instance %s from image %s/%s:%s "+
							"contains auto-scaling components, but metrics-server is not enabled in Cellery runtime. "+
							"Autoscaling may not work. ", util.Bold(subTreeRoot.Instance),
							subTreeRoot.MetaData.CellImageName.Organization,
							subTreeRoot.MetaData.CellImageName.Name, subTreeRoot.MetaData.CellImageName.Version))
					}
				} else {
					util.PrintWarningMessage("Error occurred while checking whether metrics-server is enabled.")
				}
			}
			validatedImages = append(validatedImages, subTreeRoot.Instance)

			// Traversing down the dependency tree
			for _, dependency := range subTreeRoot.Dependencies {
				if !dependency.IsRunning {
					err := validateDependencySubtreeOffline(dependency)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	err := validateDependencySubtreeOffline(treeRoot)
	if err != nil {
		return err
	}
	return nil
}

// confirmDependencyTree confirms from the user whether the intended dependency tree had been resolved
func confirmDependencyTree(tree *dependencyTreeNode, assumeYes bool) error {
	var dependencyData [][]string
	var traversedInstances []string
	// Preparing instances table data
	var extractDependencyTreeData func(subTree *dependencyTreeNode)
	extractDependencyTreeData = func(subTree *dependencyTreeNode) {
		for _, dependency := range subTree.Dependencies {
			// Traversing the dependency tree
			if !dependency.IsRunning {
				extractDependencyTreeData(dependency)
			}

			// Adding used instances table content
			instanceAlreadyAdded := false
			for _, instance := range traversedInstances {
				if instance == dependency.Instance {
					instanceAlreadyAdded = true
					break
				}
			}
			if !instanceAlreadyAdded {
				var usedInstance string
				if dependency.IsRunning {
					usedInstance = "Available in Runtime"
				} else {
					usedInstance = "To be Created"
				}
				var sharedSymbol string
				if dependency.IsShared {
					sharedSymbol = "Shared"
				} else {
					sharedSymbol = " - "
				}
				dependencyData = append(dependencyData, []string{
					dependency.Instance,
					dependency.MetaData.Organization + "/" + dependency.MetaData.Name + ":" +
						dependency.MetaData.Version,
					usedInstance,
					sharedSymbol,
				})
				traversedInstances = append(traversedInstances, dependency.Instance)
			}
		}
	}
	extractDependencyTreeData(tree)
	dependencyData = append(dependencyData, []string{
		tree.Instance,
		tree.MetaData.Organization + "/" + tree.MetaData.Name + ":" + tree.MetaData.Version,
		"To be Created",
		" - ",
	})

	// Rendering the instances table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"INSTANCE NAME", "CELL IMAGE", "USED INSTANCE", "SHARED"})
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetAlignment(3)
	table.SetRowSeparator("-")
	table.SetCenterSeparator(" ")
	table.SetColumnSeparator(" ")
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold})
	table.SetColumnColor(
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{})
	table.AppendBulk(dependencyData)
	fmt.Printf("\n%s:\n\n", util.Bold("Instances to be Used"))
	table.Render()

	// Printing the dependency tree
	fmt.Printf("\n%s:\n\n", util.Bold("Dependency Tree to be Used"))
	var printDependencyTree func(subTree *dependencyTreeNode, nestingLevel int, ancestorBranchPrintRequirement []bool)
	printDependencyTree = func(subTree *dependencyTreeNode, nestingLevel int, ancestorBranchPrintRequirement []bool) {
		var index = 0
		for alias, dependency := range subTree.Dependencies {
			// Adding the dependency tree visualization content
			for j := 0; j < nestingLevel; j++ {
				if ancestorBranchPrintRequirement[j] {
					fmt.Print("   │ ")
				} else {
					fmt.Print("     ")
				}
			}
			if index == len(subTree.Dependencies)-1 {
				fmt.Print("   └")
			} else {
				fmt.Print("   ├")
			}
			fmt.Printf("── %s: %s\n", util.Bold(alias), dependency.Instance)

			// Traversing the dependency tree
			if !dependency.IsRunning {
				printDependencyTree(dependency, nestingLevel+1,
					append(ancestorBranchPrintRequirement, index != len(subTree.Dependencies)-1))
			}
			index++
		}
	}
	if len(tree.Dependencies) > 0 {
		fmt.Printf(" %s\n", util.Bold(tree.Instance))
		printDependencyTree(tree, 0, []bool{})
	} else {
		fmt.Printf(" %s\n", util.Bold("No Dependencies"))
	}
	fmt.Println()

	if !assumeYes {
		fmt.Printf("%s Do you wish to continue with starting above Cell instances (Y/n)? ", util.YellowBold("?"))
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSpace(confirmation)) == "n" {
			return fmt.Errorf("running Cell aborted")
		}
	}
	fmt.Println()
	return nil
}

// startDependencyTree starts up the whole dependency tree except the root
// This does not start the root of the dependency tree
func startDependencyTree(registry string, tree *dependencyTreeNode, spinner *util.Spinner,
	instanceEnvVars map[string][]*environmentVariable) {
	const errorMessage = "Error occurred while starting the dependency tree"
	var wg sync.WaitGroup
	wg.Add(len(tree.Dependencies))
	for _, dependency := range tree.Dependencies {
		if dependency.IsRunning {
			wg.Done()
		} else { // This level of checking is done to not start unwanted goroutines
			go func(dependencyNode *dependencyTreeNode) {
				defer wg.Done()
				dependencyNode.Mux.Lock()
				defer dependencyNode.Mux.Unlock()
				if !dependencyNode.IsRunning { // This level of checking is done to make sure the condition is met
					startDependencyTree(registry, dependencyNode, spinner, instanceEnvVars)
					cellImage := &image.CellImage{
						Registry:     registry,
						Organization: dependencyNode.MetaData.Organization,
						ImageName:    dependencyNode.MetaData.Name,
						ImageVersion: dependencyNode.MetaData.Version,
					}
					imageDir, err := ExtractImage(cellImage, true, spinner)
					if err != nil {
						spinner.Stop(false)
						util.ExitWithErrorMessage(errorMessage, fmt.Errorf("failed to extract "+
							"cell image %s/%s:%s due to %v", dependencyNode.MetaData.Organization,
							dependencyNode.MetaData.Name, dependencyNode.MetaData.Version, err))
					}

					err = startCellInstance(imageDir, dependencyNode.Instance, dependencyNode,
						instanceEnvVars[dependencyNode.Instance])
					if err != nil {
						spinner.Stop(false)
						util.ExitWithErrorMessage(errorMessage, fmt.Errorf("failed to start "+
							"cell instance %s due to %v", dependencyNode.Instance, err))
					}
					dependencyNode.IsRunning = true
					fmt.Printf("\r\x1b[2K%s Starting instance %s\n", util.Green("\U00002714"),
						dependencyNode.Instance)

					err = os.RemoveAll(imageDir)
					if err != nil {
						spinner.Stop(false)
						util.ExitWithErrorMessage(errorMessage, fmt.Errorf("failed to cleanup due to %v", err))
					}
				}
			}(dependency)
		}
	}
	wg.Wait()
}

func startCellInstance(imageDir string, instanceName string, runningNode *dependencyTreeNode,
	envVars []*environmentVariable) error {
	imageTag := fmt.Sprintf("%s/%s:%s", runningNode.MetaData.Organization, runningNode.MetaData.Name,
		runningNode.MetaData.Version)
	balFileName, err := util.GetSourceFileName(filepath.Join(imageDir, constants.ZIP_BALLERINA_SOURCE))
	if err != nil {
		return fmt.Errorf("failed to find source file in Cell Image %s due to %v", imageTag, err)
	}
	balFilePath := filepath.Join(imageDir, constants.ZIP_BALLERINA_SOURCE, balFileName)

	containsRunFunction, err := util.RunMethodExists(balFilePath)
	if err != nil {
		return fmt.Errorf("failed to check whether run function exists in Cell Image %s due to %v", imageTag, err)
	}
	if containsRunFunction {
		// Generating the first level dependency map
		dependencies := map[string]*dependencyInfo{}
		for alias, dependency := range runningNode.Dependencies {
			dependencies[alias] = &dependencyInfo{
				Organization: dependency.MetaData.Organization,
				Name:         dependency.MetaData.Name,
				Version:      dependency.MetaData.Version,
				InstanceName: dependency.Instance,
			}
		}

		// Preparing the dependency instance map
		dependenciesJson, err := json.Marshal(dependencies)
		if err != nil {
			return fmt.Errorf("failed to start the Cell Image %s due to %v", imageTag, err)
		}
		tempRunFileName, err := util.CreateTempExecutableBalFile(balFilePath, "run")
		if err != nil {
			util.ExitWithErrorMessage("Error executing ballerina file", err)
		}
		// Preparing the run command arguments
		cmdArgs := []string{"run"}
		for _, envVar := range envVars {
			cmdArgs = append(cmdArgs, "-e", envVar.Key+"="+envVar.Value)
		}
		var imageNameStruct = &dependencyInfo{
			Organization: runningNode.MetaData.Organization,
			Name:         runningNode.MetaData.Name,
			Version:      runningNode.MetaData.Version,
			InstanceName: instanceName,
		}
		iName, err := json.Marshal(imageNameStruct)
		if err != nil {
			util.ExitWithErrorMessage("Error in generating cellery:CellImageName construct", err)
		}
		cmdArgs = append(cmdArgs, tempRunFileName, "run", string(iName), string(dependenciesJson))

		// Calling the run function
		moduleMgr := &util.BLangManager{}
		exePath, err := moduleMgr.GetExecutablePath()
		if err != nil {
			util.ExitWithErrorMessage("Failed to get executable path", err)
		}

		cmd := &exec.Cmd{}

		if exePath != "" {
			cmd = exec.Command(exePath+"ballerina", cmdArgs...)
		} else {
			currentDir, err := os.Getwd()
			if err != nil {
				util.ExitWithErrorMessage("Error in determining working directory", err)
			}

			//Retrieve the cellery cli docker instance status.
			cmdDockerPs := exec.Command("docker", "ps", "--filter", "label=ballerina-runtime="+version.BuildVersion(),
				"--filter", "label=currentDir="+currentDir, "--filter", "status=running", "--format", "{{.ID}}")

			containerId, err := cmdDockerPs.Output()
			if err != nil {
				util.ExitWithErrorMessage("Docker Run Error", err)
			}

			if string(containerId) == "" {

				cmdDockerRun := exec.Command("docker", "run", "-d", "-l", "ballerina-runtime="+version.BuildVersion(),
					"--mount", "type=bind,source="+util.UserHomeDir()+string(os.PathSeparator)+".ballerina,target=/home/cellery/.ballerina",
					"--mount", "type=bind,source="+util.UserHomeDir()+string(os.PathSeparator)+".cellery,target=/home/cellery/.cellery",
					"--mount", "type=bind,source="+util.UserHomeDir()+string(os.PathSeparator)+".kube,target=/home/cellery/.kube",
					"wso2cellery/ballerina-runtime:"+version.BuildVersion(), "sleep", "600",
				)

				containerId, err = cmdDockerRun.Output()
				if err != nil {
					util.ExitWithErrorMessage("Docker Run Error", err)
				}
				time.Sleep(5 * time.Second)
			}

			cliUser, err := user.Current()
			if err != nil {
				util.ExitWithErrorMessage("Error while retrieving the current user", err)
			}

			exeUid := constants.CELLERY_DOCKER_CLI_USER_ID

			if cliUser.Uid != constants.CELLERY_DOCKER_CLI_USER_ID && runtime.GOOS == "linux" {
				cmdUserExist := exec.Command("docker", "exec", strings.TrimSpace(string(containerId)),
					"id", "-u", cliUser.Username)
				_, errUserExist := cmdUserExist.Output()
				if errUserExist != nil {
					cmdUserAdd := exec.Command("docker", "exec", strings.TrimSpace(string(containerId)), "useradd", "-m",
						"-d", "/home/cellery", "--uid", cliUser.Uid, cliUser.Username)

					_, errUserAdd := cmdUserAdd.Output()
					if errUserAdd != nil {
						util.ExitWithErrorMessage("Error in adding Cellery execution user", errUserAdd)
					}
				}
				exeUid = cliUser.Uid
			}

			cmdArgs = append(cmdArgs, "-e", constants.CELLERY_IMAGE_DIR_ENV_VAR+"="+imageDir)

			re := regexp.MustCompile(`^.*cellery-cell-image`)
			tempRunFileName = re.ReplaceAllString(tempRunFileName, "/home/cellery/.cellery/tmp/cellery-cell-image")
			dockerImageDir := re.ReplaceAllString(imageDir, "/home/cellery/.cellery/tmp/cellery-cell-image")

			cmd = exec.Command("docker", "exec", "-e", constants.CELLERY_IMAGE_DIR_ENV_VAR+"="+dockerImageDir)
			shellEnvs := os.Environ()
			// check if any env var prepended with `CELLERY` exists. If so, set them to docker exec command.
			if len(shellEnvs) != 0 {
				for _, shellEnv := range shellEnvs {
					if strings.HasPrefix(shellEnv, "CELLERY") {
						cmd.Args = append(cmd.Args, "-e", shellEnv)
					}
				}
			}
			// set any explicitly passed env vars in cellery run command to the docker exec.
			// This will override any env vars with identical names (prefixed with 'CELLERY') set previously.
			if len(envVars) != 0 {
				for _, envVar := range envVars {
					cmd.Args = append(cmd.Args, "-e", envVar.Key+"="+envVar.Value)
				}
			}
			cmd.Args = append(cmd.Args, "-w", "/home/cellery", "-u", exeUid,
				strings.TrimSpace(string(containerId)), constants.DOCKER_CLI_BALLERINA_EXECUTABLE_PATH, "run", tempRunFileName, "run",
				string(iName), string(dependenciesJson))
		}
		defer os.Remove(imageDir)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, constants.CELLERY_IMAGE_DIR_ENV_VAR+"="+imageDir)
		stdoutReader, _ := cmd.StdoutPipe()
		stdoutScanner := bufio.NewScanner(stdoutReader)
		go func() {
			for stdoutScanner.Scan() {
				fmt.Printf("\r\x1b[2K\033[36m%s\033[m\n", stdoutScanner.Text())
			}
		}()
		stderrReader, _ := cmd.StderrPipe()
		stderrScanner := bufio.NewScanner(stderrReader)
		go func() {
			for stderrScanner.Scan() {
				fmt.Printf("\r\x1b[2K\033[36m%s\033[m\n", stderrScanner.Text())
			}
		}()
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to execute run method in Cell instance %s due to %v", instanceName, err)
		}
		err = cmd.Wait()
		if err != nil {
			return fmt.Errorf("failed to execute run method in Cell instance %s due to %v", instanceName, err)
		}
		_ = os.Remove(tempRunFileName)
	}

	// Update the Cell instance name
	celleryDir := filepath.Join(imageDir, constants.ZIP_ARTIFACTS, "cellery")
	k8sYamlFile := filepath.Join(celleryDir, runningNode.MetaData.Name+".yaml")
	if instanceName != "" {
		// Cell instance name changed.
		err = util.ReplaceInstanceName(k8sYamlFile, fmt.Sprintf("  name: \"%s\"", runningNode.MetaData.Name), "  name: "+instanceName)
		if err != nil {
			return fmt.Errorf("failed to set instance name %s due to %v", instanceName, err)
		}
	}

	// Applying the yaml
	cellYamls, err := getYamlFiles(celleryDir)
	if err != nil {
		return fmt.Errorf("failed to find yaml files in directory %s due to %v", celleryDir, err)
	}
	for _, v := range cellYamls {
		err := kubectl.ApplyFile(v)
		if err != nil {
			return fmt.Errorf("failed to create k8s artifacts %s from image %s due to %v", v, instanceName, err)
		}
	}

	// Waiting for the Cell to be Ready
	err = kubectl.WaitForCondition("Ready", 30*60, "cells.mesh.cellery.io/"+instanceName,
		"default")
	if err != nil {
		return fmt.Errorf("error waiting for instance %s to be ready: %v", instanceName, err)
	}
	return nil
}

// generateRandomInstanceName generates a random instance name with a UUID as the suffix
func generateRandomInstanceName(dependencyMetaData *image.MetaData) (string, error) {
	u := make([]byte, 4)
	_, err := rand.Read(u)
	if err != nil {
		return "", err
	}
	uuid := fmt.Sprintf("%x", u)

	// Generating random instance name
	return dependencyMetaData.Name + "-" + strings.Replace(dependencyMetaData.Version, ".", "-", -1) + "-" + uuid, nil
}

// Get list of yaml files in a dir.
func getYamlFiles(path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if filepath.Ext(path) == ".yaml" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// extractImage extracts the image into a temporary directory and returns the path.
// Cleaning the path after finishing your work is your responsibility.
func ExtractImage(cellImage *image.CellImage, pullIfNotPresent bool, spinner *util.Spinner) (string, error) {
	repoLocation := filepath.Join(util.UserHomeDir(), constants.CELLERY_HOME, "repo", cellImage.Organization,
		cellImage.ImageName, cellImage.ImageVersion)
	zipLocation := filepath.Join(repoLocation, cellImage.ImageName+constants.CELL_IMAGE_EXT)

	// Pull image if not exist
	imageExists, err := util.FileExists(zipLocation)
	if err != nil {
		return "", err
	}
	if !imageExists {
		if pullIfNotPresent {
			if spinner != nil {
				spinner.Pause()
			}
			cellImageTag := cellImage.Registry + "/" + cellImage.Organization + "/" + cellImage.ImageName +
				":" + cellImage.ImageVersion
			RunPull(cellImageTag, true, "", "")
			fmt.Println()
			if spinner != nil {
				spinner.Resume()
			}
		} else {
			return "", fmt.Errorf("image %s/%s:%s not present on the local repository", cellImage.Organization,
				cellImage.ImageName, cellImage.ImageVersion)
		}
	}

	// Unzipping image to a temporary location
	celleryHomeTmp := path.Join(util.UserHomeDir(), constants.CELLERY_HOME, "tmp")
	if _, err := os.Stat(celleryHomeTmp); os.IsNotExist(err) {
		os.Mkdir(celleryHomeTmp, 0755)
	}

	tempPath, err := ioutil.TempDir(celleryHomeTmp, "cellery-cell-image")
	if err != nil {
		return "", err
	}
	err = util.Unzip(zipLocation, tempPath)
	if err != nil {
		return "", nil
	}
	return tempPath, nil
}

// dependencyAliasLink is used to store the link information provided by the user
type dependencyAliasLink struct {
	Instance           string
	DependencyAlias    string
	DependencyInstance string
	IsRunning          bool
}

// environmentVariable is used to store the environment variables to be passed to the instances
type environmentVariable struct {
	Key   string
	Value string
}

// dependencyTreeNode is used as a node of the dependency tree
type dependencyTreeNode struct {
	Mux          sync.Mutex
	Instance     string
	MetaData     *image.MetaData
	Dependencies map[string]*dependencyTreeNode
	IsShared     bool
	IsRunning    bool
}

// dependencyInfo is used to pass the dependency information to Ballerina
type dependencyInfo struct {
	Organization string `json:"org"`
	Name         string `json:"name"`
	Version      string `json:"ver"`
	InstanceName string `json:"instanceName"`
}
