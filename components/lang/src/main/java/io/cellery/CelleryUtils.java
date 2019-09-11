/*
 * Copyright (c) 2018, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
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
package io.cellery;

import io.cellery.models.API;
import io.cellery.models.Cell;
import io.cellery.models.Component;
import io.cellery.models.OIDC;
import io.cellery.models.Test;
import io.cellery.models.Web;
import io.fabric8.kubernetes.api.model.HTTPGetActionBuilder;
import io.fabric8.kubernetes.api.model.HTTPHeader;
import io.fabric8.kubernetes.api.model.HTTPHeaderBuilder;
import io.fabric8.kubernetes.api.model.Probe;
import io.fabric8.kubernetes.api.model.ProbeBuilder;
import io.fabric8.kubernetes.api.model.Quantity;
import io.fabric8.kubernetes.api.model.ResourceRequirements;
import io.fabric8.kubernetes.client.utils.Serialization;
import org.apache.commons.io.FileUtils;
import org.apache.commons.lang3.StringUtils;
import org.ballerinalang.model.values.BInteger;
import org.ballerinalang.model.values.BMap;
import org.ballerinalang.model.values.BString;
import org.ballerinalang.model.values.BValue;
import org.ballerinalang.model.values.BValueArray;
import org.ballerinalang.util.exceptions.BallerinaException;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.function.Consumer;
import java.util.stream.Collectors;
import java.util.stream.IntStream;

import static io.cellery.CelleryConstants.DEFAULT_PARAMETER_VALUE;
import static io.cellery.CelleryConstants.KIND;
import static io.cellery.CelleryConstants.LIMITS;
import static io.cellery.CelleryConstants.LIVENESS;
import static io.cellery.CelleryConstants.READINESS;
import static io.cellery.CelleryConstants.REQUESTS;
import static io.cellery.CelleryConstants.RESOURCES;
import static io.cellery.CelleryConstants.TARGET;

/**
 * Cellery Utility methods.
 */
public class CelleryUtils {

    /**
     * Returns swagger file as a String.
     *
     * @param path     swagger file path
     * @param encoding string encoding
     * @return swagger file as a String
     * @throws IOException if unable to read file
     */
    public static String readSwaggerFile(String path, Charset encoding) throws IOException {
        byte[] encoded = Files.readAllBytes(Paths.get(path));
        return new String(encoded, encoding);
    }

    /**
     * Returns valid kubernetes name.
     *
     * @param name actual value
     * @return valid name
     */
    public static String getValidName(String name) {
        return name.toLowerCase(Locale.getDefault()).replaceAll("\\P{Alnum}", "-");
    }


    /**
     * Process Web Ingress and add to component.
     *
     * @param component    Component
     * @param attributeMap WebIngress properties
     */
    public static void processWebIngress(Component component, LinkedHashMap attributeMap) {
        Web webIngress = new Web();
        LinkedHashMap gatewayConfig = ((BMap) attributeMap.get("gatewayConfig")).getMap();
        API httpAPI = getApi(component, attributeMap);
        httpAPI.setGlobal(true);
        httpAPI.setBackend(component.getService());
        httpAPI.setContext(((BString) gatewayConfig.get("context")).stringValue());
        webIngress.setHttpAPI(httpAPI);
        webIngress.setVhost(((BString) gatewayConfig.get("vhost")).stringValue());
        if (gatewayConfig.containsKey("tls")) {
            // TLS enabled
            LinkedHashMap tlsConfig = ((BMap) gatewayConfig.get("tls")).getMap();
            webIngress.setTlsKey(((BString) tlsConfig.get("key")).stringValue());
            webIngress.setTlsCert(((BString) tlsConfig.get("cert")).stringValue());
            if (StringUtils.isBlank(webIngress.getTlsKey())) {
                printWarning("TLS Key value is empty in component " + component.getName());
            }
            if (StringUtils.isBlank(webIngress.getTlsCert())) {
                printWarning("TLS Cert value is empty in component " + component.getName());
            }
        }
        if (gatewayConfig.containsKey("oidc")) {
            // OIDC enabled
            webIngress.setOidc(processOidc(((BMap) gatewayConfig.get("oidc")).getMap()));
        }
        component.addWeb(webIngress);
    }

    /**
     * Process API info and returns a API.
     *
     * @param component    component object
     * @param attributeMap API attribute map
     * @return API object
     */
    public static API getApi(Component component, LinkedHashMap attributeMap) {
        API httpAPI = new API();
        int containerPort = (int) ((BInteger) attributeMap.get("port")).intValue();
        // Validate the container port is same for all the ingresses.
        if (component.getContainerPort() > 0 && containerPort != component.getContainerPort()) {
            throw new BallerinaException("Invalid container port" + containerPort + ". Multiple container ports are " +
                    "not supported.");
        }
        component.setContainerPort(containerPort);
        return httpAPI;
    }

    /**
     * Extract the Readiness Probe & Liveness Probe.
     *
     * @param probes    Scale policy to be processed
     * @param component current component
     */
    public static void processProbes(LinkedHashMap<?, ?> probes, Component component) {
        if (probes.containsKey(LIVENESS)) {
            LinkedHashMap livenessConf = ((BMap) probes.get(LIVENESS)).getMap();
            component.setLivenessProbe(getProbe(livenessConf));
        }
        if (probes.containsKey(READINESS)) {
            LinkedHashMap readinessConf = ((BMap) probes.get(READINESS)).getMap();
            component.setReadinessProbe(getProbe(readinessConf));
        }
    }

    /**
     * Extract the Resource limits and requests.
     *
     * @param resources Resource to be processed
     * @param component current component
     */
    public static void processResources(LinkedHashMap<?, ?> resources, Component component) {
        ResourceRequirements resourceRequirements = new ResourceRequirements();
        if (resources.containsKey(LIMITS)) {
            LinkedHashMap limitsConf = ((BMap) resources.get(LIMITS)).getMap();
            resourceRequirements.setLimits(getResourceQuantityMap(limitsConf));
        }
        if (resources.containsKey(REQUESTS)) {
            LinkedHashMap requestConf = ((BMap) resources.get(REQUESTS)).getMap();
            resourceRequirements.setRequests(getResourceQuantityMap(requestConf));
        }
        component.setResources(resourceRequirements);
    }

    /**
     * Get Resource Quantity Map.
     *
     * @param conf map of configurations
     * @return ResourceQuantityMap
     */
    private static Map<String, Quantity> getResourceQuantityMap(LinkedHashMap<String, BValue> conf) {
        return conf.entrySet()
                .stream()
                .collect(Collectors.toMap(Map.Entry::getKey,
                        e -> new Quantity(e.getValue().stringValue()))
                );
    }

    /**
     * Create ProbeBuilder with given Liveness/Readiness Probe config.
     *
     * @param probeConf probeConfig map
     * @return ProbeBuilder
     */
    private static Probe getProbe(LinkedHashMap probeConf) {
        ProbeBuilder probeBuilder = new ProbeBuilder();
        final BMap probeKindMap = (BMap) probeConf.get(KIND);
        LinkedHashMap probeKindConf = probeKindMap.getMap();
        String probeKind = probeKindMap.getType().getName();
        if ("TcpSocket".equals(probeKind)) {
            probeBuilder.withNewTcpSocket()
                    .withNewPort((int) ((BInteger) probeKindConf.get("port")).intValue())
                    .endTcpSocket();
        } else if ("HttpGet".equals(probeKind)) {
            List<HTTPHeader> headers = new ArrayList<>();
            if (probeKindConf.containsKey("httpHeaders")) {
                ((BMap<?, ?>) probeKindConf.get("httpHeaders")).getMap().forEach((key, value) -> {
                    HTTPHeader header = new HTTPHeaderBuilder()
                            .withName(key.toString())
                            .withValue(value.stringValue())
                            .build();
                    headers.add(header);
                });
            }
            probeBuilder.withHttpGet(new HTTPGetActionBuilder()
                    .withNewPort((int) ((BInteger) probeKindConf.get("port")).intValue())
                    .withPath(((BString) probeKindConf.get("path")).stringValue())
                    .withHttpHeaders(headers)
                    .build()
            );
        } else {
            final BValueArray commandList = (BValueArray) probeKindConf.get("commands");
            String[] commands = Arrays.copyOfRange(commandList.getStringArray(), 0, (int) commandList.size());
            probeBuilder.withNewExec().addToCommand(commands).endExec();
        }
        return probeBuilder
                .withInitialDelaySeconds((int) (((BInteger) probeConf.get("initialDelaySeconds")).intValue()))
                .withPeriodSeconds((int) (((BInteger) probeConf.get("periodSeconds")).intValue()))
                .withFailureThreshold((int) (((BInteger) probeConf.get("failureThreshold")).intValue()))
                .withTimeoutSeconds((int) (((BInteger) probeConf.get("timeoutSeconds")).intValue()))
                .withSuccessThreshold((int) (((BInteger) probeConf.get("successThreshold")).intValue())).build();
    }


    /**
     * Process envVars and add to component.
     *
     * @param envVars Map of EnvVars
     * @param test    targetComponent
     */
    public static void processEnvVars(LinkedHashMap<?, ?> envVars, Test test) {
        envVars.forEach((k, v) -> {
            if (((BMap) v).getMap().get("value").toString().isEmpty()) {
                //value is empty for envVar
                test.addEnv(k.toString(), DEFAULT_PARAMETER_VALUE);
            } else {
                test.addEnv(k.toString(), ((BMap) v).getMap().get("value").toString());
            }
        });
    }

    /**
     * Process envVars and add to test.
     *
     * @param envVars   Map of EnvVars
     * @param component targetComponent
     */
    public static void processEnvVars(LinkedHashMap<?, ?> envVars, Component component) {
        envVars.forEach((k, v) -> {
            if (((BMap) v).getMap().get("value").toString().isEmpty()) {
                //value is empty for envVar
                component.addEnv(k.toString(), DEFAULT_PARAMETER_VALUE);
            } else {
                component.addEnv(k.toString(), ((BMap) v).getMap().get("value").toString());
            }
        });
    }

    /**
     * Process OIDCConfig.
     *
     * @param oidcConfig OIDC configuration
     */
    private static OIDC processOidc(LinkedHashMap oidcConfig) {
        OIDC oidc = new OIDC();
        oidc.setProviderUrl(((BString) oidcConfig.get("providerUrl")).stringValue());
        oidc.setRedirectUrl(((BString) oidcConfig.get("redirectUrl")).stringValue());
        oidc.setBaseUrl(((BString) oidcConfig.get("baseUrl")).stringValue());
        oidc.setClientId(((BString) oidcConfig.get("clientId")).stringValue());
        BValueArray nonSecurePaths = ((BValueArray) oidcConfig.get("nonSecurePaths"));
        Set<String> nonSecurePathList = new HashSet<>();
        IntStream.range(0, (int) nonSecurePaths.size()).forEach(nonSecurePathIndex ->
                nonSecurePathList.add(nonSecurePaths.getString(nonSecurePathIndex)));
        oidc.setNonSecurePaths(nonSecurePathList);

        BValueArray securePaths = ((BValueArray) oidcConfig.get("securePaths"));
        Set<String> securePathList = new HashSet<>();
        IntStream.range(0, (int) securePaths.size()).forEach(securePathIndex ->
                securePathList.add(securePaths.getString(securePathIndex)));
        oidc.setSecurePaths(securePathList);

        if (((BValue) oidcConfig.get("clientSecret")).getType().getName().equals("string")) {
            // Not using DCR
            oidc.setClientSecret(((BString) oidcConfig.get("clientSecret")).stringValue());
        } else {
            // Using DCR
            LinkedHashMap dcrConfig = ((BMap) oidcConfig.get("clientSecret")).getMap();
            oidc.setDcrUser(((BString) dcrConfig.get("dcrUser")).stringValue());
            oidc.setDcrPassword(((BString) dcrConfig.get("dcrPassword")).stringValue());
            if (dcrConfig.containsKey("dcrUrl")) {
                // DCR url is optional
                oidc.setDcrUrl(((BString) oidcConfig.get("dcrUrl")).stringValue());
            }
        }
        if (oidcConfig.containsKey("subjectClaim")) {
            //optional field
            oidc.setSubjectClaim(((BString) oidcConfig.get("subjectClaim")).stringValue());
        }
        return oidc;
    }

    /**
     * Write content to a File. Create the required directories if they don't not exists.
     *
     * @param context    context of the file
     * @param targetPath target file path
     * @throws IOException If an error occurs when writing to a file
     */
    public static void writeToFile(String context, String targetPath) throws IOException {
        File newFile = new File(targetPath);
        // delete if file exists
        if (newFile.exists()) {
            Files.delete(Paths.get(newFile.getPath()));
        }
        //create required directories
        if (newFile.getParentFile().mkdirs()) {
            Files.write(Paths.get(targetPath), context.getBytes(StandardCharsets.UTF_8));
            return;
        }
        Files.write(Paths.get(targetPath), context.getBytes(StandardCharsets.UTF_8));
    }

    /**
     * Generates Yaml from a object.
     *
     * @param object Object
     * @param <T>    Any Object type
     * @return Yaml as a string.
     */
    public static <T> String toYaml(T object) {
        return Serialization.asYaml(object);
    }

    /**
     * Print a Warning message.
     *
     * @param message warning message
     */
    public static void printWarning(String message) {
        PrintStream out = System.out;
        out.println("Warning: " + message);
    }

    /**
     * Print a Info message.
     *
     * @param message info message
     */
    public static void printInfo(String message) {
        PrintStream out = System.out;
        out.println("Info: " + message);
    }

    /**
     * Print a Debug message.
     *
     * @param message debug message
     */
    public static void printDebug(String message) {
        if ("true".equalsIgnoreCase(System.getenv("DEBUG_MODE"))) {
            PrintStream out = System.out;
            out.println("Debug: " + message);
        }
    }

    /**
     * Copy file target/resources directory.
     *
     * @param sourcePath source file/directory path
     */
    public static void copyResourceToTarget(String sourcePath) {
        File src = new File(sourcePath);
        String targetPath = TARGET + File.separator + RESOURCES + File.separator + src.getName();
        File dst = new File(targetPath);
        // if source is file
        try {
            if (Files.isRegularFile(Paths.get(sourcePath))) {
                if (Files.isDirectory(dst.toPath())) {
                    // if destination is directory
                    FileUtils.copyFileToDirectory(src, dst);
                } else {
                    // if destination is file
                    FileUtils.copyFile(src, dst);
                }
            } else if (Files.isDirectory(Paths.get(sourcePath))) {
                FileUtils.copyDirectory(src, dst);
            }

        } catch (IOException e) {
            throw new BallerinaException("Error occurred while copying resource file " + sourcePath +
                    ". " + e.getMessage());
        }
    }

    /**
     * Executes a shell command.
     *
     * @param command          command to execute
     * @param workingDirectory working directory
     * @param stdout           stdout of the command
     * @param stderr           stderr of the command
     * @return stdout/stderr
     */
    public static String executeShellCommand(String command, Path workingDirectory, Writer stdout, Writer stderr) {
        StringBuilder stdOut = new StringBuilder();
        StringBuilder stdErr = new StringBuilder();
        ProcessBuilder processBuilder = new ProcessBuilder("/bin/bash", "-c", command);

        ExecutorService executor = Executors.newFixedThreadPool(2);
        int exitCode;
        try {
            if (workingDirectory != null) {
                File workDirectory = workingDirectory.toFile();
                if (workDirectory.exists()) {
                    processBuilder.directory(workDirectory);
                }
            }
            Process process = processBuilder.start();

            StreamGobbler outputStreamGobbler = new StreamGobbler(process.getInputStream(), msg -> {
                stdOut.append(msg);
                stdout.writeMessage(msg);
            });
            StreamGobbler errorStreamGobbler = new StreamGobbler(process.getErrorStream(), msg -> {
                stdErr.append(msg);
                stderr.writeMessage(msg);
            });

            executor.execute(outputStreamGobbler);
            executor.execute(errorStreamGobbler);

            exitCode = process.waitFor();
            if (exitCode > 0) {
                throw new BallerinaException("Command " + command + " exited with exit code " + exitCode);
            }

        } catch (IOException e) {
            throw new BallerinaException(
                    "Error occurred while executing the command '" + command + "', " + "from directory '"
                            + workingDirectory.toString(), e);
        } catch (InterruptedException e) {
            throw new BallerinaException(
                    "InterruptedException occurred while executing the command '" + command + "', " + "from directory '"
                            + workingDirectory.toString(), e);
        } finally {
            executor.shutdownNow();
        }

        if (stdOut.toString().isEmpty()) {
            return stdErr.toString();
        }
        return stdOut.toString();
    }

    /**
     * Read the yaml and create a Cell object.
     *
     * @param destinationPath YAML path
     * @return Constructed Cell object
     */
    public static Cell readCellYaml(String destinationPath) {
        Cell cell;
        try (FileInputStream fileInputStream = new FileInputStream(destinationPath)) {
            cell = Serialization.unmarshal(fileInputStream, Cell.class);
        } catch (IOException e) {
            throw new BallerinaException("Unable to read Cell image file " + destinationPath + ". \nDid you " +
                    "pull/build the cell image ?");
        }
        if (cell == null) {
            throw new BallerinaException("Unable to extract Cell image from YAML " + destinationPath);
        }
        return cell;
    }

    /**
     * Interface to print shell command output.
     */
    public interface Writer {

        /**
         * Called when a newline should be printed.
         *
         * @param msg message to write
         */
        void writeMessage(String msg);
    }

    /**
     * StreamGobbler to handle process builder output.
     */
    private static class StreamGobbler implements Runnable {
        private InputStream inputStream;
        private Consumer<String> consumer;

        StreamGobbler(InputStream inputStream, Consumer<String> consumer) {
            this.inputStream = inputStream;
            this.consumer = consumer;
        }

        @Override
        public void run() {
            new BufferedReader(new InputStreamReader(inputStream, StandardCharsets.UTF_8)).lines()
                    .forEach(consumer);
        }
    }
}
