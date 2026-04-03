# Comparison tests

For the sake of clarity we define node $A$ the one that runs the *control-points*, node $B$ the one running the *devices* and $C$ the one running the MQTT Broker.

This distinction does not imply that node $A$, $B$ and $C$ can not be the same node, and running the test in local.

## 💠 Prerequisites

### Firewall rules

* Node $A$ should be able to connect via **ssh** to node $B$.
* Nodes $A$ and $B$ should be able to send and receive data between each other.
* Nodes $A$ and $B$ should be able to send and receive data on any **UDP** and **TCP** port.
* Node $A$ and $B$ should able to communicate to the TCP port **1883** to node $C$.

### SSH

Node $A$ should be able to communicate with node $B$ via **ssh password-less**.

> [!NOTE]
>
> To speed-up the testing time node $A$ should add to <code>~/ssh/config</code> the following lines:

```
Host *
    # Enable Multiplexing
    ControlMaster auto

    # Use a unique path with variables
    # %r = remote username, %h = host, %p = port
    ControlPath ~/.ssh/sockets/%r@%h:%p

    # Keep the master connection open for 10 minutes after you close your last session
    ControlPersist 10m

    # Keep-alive settings (prevents timeouts on flaky Wi-Fi)
    ServerAliveInterval 60
    ServerAliveCountMax 3

    Compression yes
```

> [!IMPORTANT]
>
> The folder <code>~/.ssh/sockets</code> should be **created manually**, if it does not already exists.



## 💠Run tests

> [!CAUTION]
>
> Be sure node $B$ (and $C$ for MQTT-D) are up and running, no explicit error will be prompted in case of failure.
>
> Every log information can be found in <code>tests/logs</code>.

> [!WARNING]
>
> Running multiple test will not override the content of the folder <code>tests/logs</code> but it will be more difficult to distinguish each log file which test it belongs to using only the timestamp.



1. Inside the folder <code>tests/scripts</code> add execution permission to the tests scripts:

   ```sh
   chmod +x *
   ```

2. Run tests:

   * For **UPnP**:

     > [!NOTE]
     >
     > This will also run the test of **GENA**.

     ```
     ./upnp_latency_control.sh
     ```

     

   * For **MQTT-D**:

     ```sh
     ./mqtt_latency_control.sh
     ```

> [!TIP]
>
> For your information the UPnP lasts about **2.5 days** and the MQTT-D **14 days**!



## 💠Analyse the results

To analyse the experiments results a python [notebook](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/tests/analysis/data_analysis.ipynb) is provided in <code>tests/analysis/data_analysis.ipynb</code>.

> [!NOTE]
>
> The code is not well optimised so it requires for the MQTT-D log files at least 5GB of free memory otherwise the python kernel will stop.