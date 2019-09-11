Local setup
---
This will setup the local environment, by creating a virtual machine with pre-installed kubeadm and cellery runtime. 

## Interactive Method

   i. Execute `cellery setup` command to configure Cellery runtime. This 
    will prompt a list of selections. By selecting `create ` section users can setup the Cellery runtime: 
   ```
    $ cellery setup
    [Use arrow keys]
    ? Setup Cellery runtime
        Manage
      ➤ Create
        Modify
        Switch
        EXIT
   ```

   ii. From the selections available for environment type, select `Local` to proceed with local installation:
   ```
    $ ✔ Create
    [Use arrow keys]
    ? Select an environment to be installed
      ➤ Local
        GCP
        Existing cluster
        BACK
   ```
    
   iii. For the local setup, select the preferred installation type; `Basic` or `Complete`: 
   ```
    $ ✔ Create
      ✔ Local
    [Use arrow keys]
    ? Select the type of runtime
      ➤ Basic (size: 2.17 GB)
        Complete (size: 3.74 GB)
   ```

   iv. Next, the CLI will prompt for confirmation to download and install the local setup:
   ```
    ✔ Create
    ✔ Local
    ✔ Basic (size: 2.17 GB)
    Use the arrow keys to navigate: ↓ ↑ → ← 
    ? Downloading cellery-runtime-basic-0.2.0.tar.gz will take 2.17 GB from your machine. Do you want to continue: 
      ▸ Yes
        No
   ```
   
   v. [Configure host entries](#configure-host-entries) once the cellery system is installed. 

## Inline Method

With the non-interactive method, creating the local setup with CLI commands with one go is supported. This supports both complete and basic setups as shown below:

| Setup Option | Package | Command <img width=1100/> | Description |
|--------------|------|---------|-------------|
| Local | Basic | `cellery setup create local -y` | Creates basic local setup. This download a VM and installs to your machine. You will require Virtual Box as pre-requisite to perform this operation| 
| Local | Complete | `cellery setup create local --complete -y ` | Creates complete local setup. This download a VM with complete cellery runtime and installs to your machine. You will require Virtual Box as pre-requisite to perform this operation| 

[Configure host entries](#configure-host-entries) once the cellery system is installed. 

## Configure host entries

Add below to /etc/host entries to access cellery hosts.
```
  192.168.56.10 wso2-apim cellery-dashboard wso2sp-observability-api wso2-apim-gateway cellery-k8s-metrics idp.cellery-system pet-store.com hello-world.com my-hello-world.com
```

## Trying Out

Once the installation process is completed, you can try out [quick start with cellery](../../README.md#quickstart-guide).

## Cleaning Up

Please refer readme for [managing cellery runtimes](./manage-setup.md) for details on how to clean up the setup.

## What's Next?

- [Developing a Cell](../writing-a-cell.md) - step by step explanation on how you could define your own cells.
- [Cell Specification](https://github.com/wso2-cellery/spec/blob/master/README.md) - key concepts of Cellery.
- [How to code cells?](../cellery-syntax.md) - explains how Cellery cells are written.
- [CLI commands](../cli-reference.md) - reference for CLI commands.
- [Samples](https://github.com/wso2-cellery/samples/tree/master) - a collection of useful samples.