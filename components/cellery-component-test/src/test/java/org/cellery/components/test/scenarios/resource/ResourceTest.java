/*
 * Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
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

package org.cellery.components.test.scenarios.resource;

import io.cellery.CelleryUtils;
import io.cellery.models.API;
import io.cellery.models.Cell;
import io.cellery.models.ServiceTemplate;
import io.cellery.models.ServiceTemplateSpec;
import io.fabric8.kubernetes.api.model.Container;
import io.fabric8.kubernetes.api.model.ObjectMeta;
import io.fabric8.kubernetes.api.model.Quantity;
import io.fabric8.kubernetes.api.model.ResourceRequirements;
import org.ballerinax.kubernetes.exceptions.KubernetesPluginException;
import org.ballerinax.kubernetes.utils.KubernetesUtils;
import org.cellery.components.test.models.CellImageInfo;
import org.cellery.components.test.utils.LangTestUtils;
import org.testng.Assert;
import org.testng.annotations.AfterClass;
import org.testng.annotations.Test;

import java.io.File;
import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.cellery.components.test.utils.CelleryTestConstants.ARTIFACTS;
import static org.cellery.components.test.utils.CelleryTestConstants.BAL;
import static org.cellery.components.test.utils.CelleryTestConstants.CELLERY;
import static org.cellery.components.test.utils.CelleryTestConstants.CELLERY_IMAGE_NAME;
import static org.cellery.components.test.utils.CelleryTestConstants.CELLERY_IMAGE_ORG;
import static org.cellery.components.test.utils.CelleryTestConstants.CELLERY_IMAGE_VERSION;
import static org.cellery.components.test.utils.CelleryTestConstants.CELLERY_MESH_VERSION;
import static org.cellery.components.test.utils.CelleryTestConstants.TARGET;
import static org.cellery.components.test.utils.CelleryTestConstants.YAML;

public class ResourceTest {

    private static final Path SAMPLE_DIR = Paths.get(System.getProperty("sample.dir"));
    private static final Path SOURCE_DIR_PATH = SAMPLE_DIR.resolve("resource");
    private static final Path TARGET_PATH = SOURCE_DIR_PATH.resolve(TARGET);
    private static final Path CELLERY_PATH = TARGET_PATH.resolve(CELLERY);
    private Cell cell;
    private Cell runtimeCell;
    private CellImageInfo cellImageInfo = new CellImageInfo("myorg", "stock", "1.0.0", "stock-inst");

    @Test(groups = "build")
    public void compileCellBuild() throws IOException, InterruptedException {
        Assert.assertEquals(LangTestUtils.compileCellBuildFunction(SOURCE_DIR_PATH, "resource" + BAL,
                cellImageInfo)
                , 0);
        File artifactYaml = CELLERY_PATH.resolve(cellImageInfo.getName() + YAML).toFile();
        Assert.assertTrue(artifactYaml.exists());
        cell = CelleryUtils.readCellYaml(CELLERY_PATH.resolve(cellImageInfo.getName() + YAML).toString());
    }

    @Test(groups = "build")
    public void validateBuildTimeCellAvailability() {
        Assert.assertNotNull(cell);
    }

    @Test(groups = "build")
    public void validateBuildTimeAPIVersion() {
        Assert.assertEquals(cell.getApiVersion(), "mesh.cellery.io/v1alpha1");
    }

    @Test(groups = "build")
    public void validateBuildTimeMetaData() {
        Assert.assertEquals(cell.getMetadata().getName(), cellImageInfo.getName());
        Assert.assertEquals(cell.getMetadata().getAnnotations().get(CELLERY_IMAGE_ORG), cellImageInfo.getOrg());
        Assert.assertEquals(cell.getMetadata().getAnnotations().get(CELLERY_IMAGE_NAME), cellImageInfo.getName());
        Assert.assertEquals(cell.getMetadata().getAnnotations().get(CELLERY_IMAGE_VERSION), cellImageInfo.getVer());
    }

    @Test(groups = "build")
    public void validateBuildTimeGatewayTemplate() {
        final API api = cell.getSpec().getGatewayTemplate().getSpec().getHttp().get(0);
        Assert.assertEquals(api.getBackend(), "stock");
        Assert.assertEquals(api.getContext(), "stock");
        Assert.assertEquals(api.getDefinitions().get(0).getMethod(), "GET");
        Assert.assertEquals(api.getDefinitions().get(0).getPath(), "/options");
    }

    @Test(groups = "build")
    public void validateBuildTimeServiceTemplates() {
        Assert.assertEquals(cell.getSpec().getServicesTemplates().get(0).getMetadata().getName(), "stock");
        final ServiceTemplateSpec spec = cell.getSpec().getServicesTemplates().get(0).getSpec();
        final Container container = spec.getContainer();
        Assert.assertEquals(container.getImage(), "wso2cellery/sampleapp-stock:0.3.0");
        Assert.assertEquals(container.getPorts().get(0).getContainerPort().intValue(), 8080);
        final ResourceRequirements resources = spec.getContainer().getResources();
        Assert.assertEquals(resources.getLimits().get("cpu"), new Quantity("500m"));
        Assert.assertEquals(resources.getLimits().get("memory"), new Quantity("128Mi"));
        Assert.assertEquals(resources.getRequests().get("memory"), new Quantity("64Mi"));
        Assert.assertEquals(resources.getRequests().get("cpu"), new Quantity("250m"));
        Assert.assertEquals(spec.getReplicas(), 1);
        Assert.assertEquals(spec.getServicePort(), 80);
    }

    @Test(groups = "run")
    public void compileCellRun() throws IOException, InterruptedException {
        Map<String, CellImageInfo> dependencyCells = new HashMap<>();
        String tmpDir = LangTestUtils.createTempImageDir(SOURCE_DIR_PATH, cellImageInfo.getName());
        Path tempPath = Paths.get(tmpDir);
        Assert.assertEquals(LangTestUtils.compileCellRunFunction(SOURCE_DIR_PATH, "resource" + BAL,
                cellImageInfo, dependencyCells, tmpDir), 0);
        File newYaml = tempPath.resolve(ARTIFACTS).resolve(CELLERY).resolve(cellImageInfo.getName() + YAML).toFile();
        runtimeCell = CelleryUtils.readCellYaml(newYaml.getAbsolutePath());
    }

    @Test(groups = "run")
    public void validateRunTimeCellAvailability() {
        Assert.assertNotNull(runtimeCell);
    }

    @Test(groups = "run")
    public void validateRunTimeAPIVersion() {
        Assert.assertEquals(runtimeCell.getApiVersion(), CELLERY_MESH_VERSION);
    }

    @Test(groups = "run")
    public void validateRunTimeMetaData() {
        final ObjectMeta metadata = runtimeCell.getMetadata();
        Assert.assertEquals(metadata.getName(), cellImageInfo.getName());
        Assert.assertEquals(metadata.getAnnotations().get(CELLERY_IMAGE_ORG), cellImageInfo.getOrg());
        Assert.assertEquals(metadata.getAnnotations().get(CELLERY_IMAGE_NAME), cellImageInfo.getName());
        Assert.assertEquals(metadata.getAnnotations().get(CELLERY_IMAGE_VERSION), cellImageInfo.getVer());
    }

    @Test(groups = "run")
    public void validateRunTimeServiceTemplates() {
        final List<ServiceTemplate> servicesTemplates = runtimeCell.getSpec().getServicesTemplates();
        Assert.assertEquals(servicesTemplates.get(0).getMetadata().getName(), "stock");
        final ServiceTemplateSpec spec = servicesTemplates.get(0).getSpec();
        Assert.assertEquals(spec.getReplicas(), 1);
        Assert.assertEquals(spec.getServicePort(), 80);
        final ResourceRequirements resources = spec.getContainer().getResources();
        Assert.assertEquals(resources.getLimits().get("cpu"), new Quantity("500m"));
        Assert.assertEquals(resources.getLimits().get("memory"), new Quantity("128Mi"));
        Assert.assertEquals(resources.getRequests().get("memory"), new Quantity("256Mi"));
        Assert.assertEquals(resources.getRequests().get("cpu"), new Quantity("256m"));
    }

    @AfterClass
    public void cleanUp() throws KubernetesPluginException {
        KubernetesUtils.deleteDirectory(TARGET_PATH);
    }
}

