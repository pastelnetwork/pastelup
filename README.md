# Pastelup

[![PastelNetwork](https://circleci.com/gh/pastelnetwork/pastelup.svg?style=shield)](https://app.circleci.com/pipelines/github/pastelnetwork/pastelup)

`pastelup` is a utility that can install, initialize, start, updatr and monitor Pastel Network `node`, supernode` and `walletnode`.

## Build

#### 1. Install the latest version of golang (1.17 or higher)

First, remove existing versions of golang as follows:
``` shell
sudo apt-get remove --auto-remove golang-go
sudo rm -rvf /usr/local/go
```

Then, download and install golang as follows:

``` shell
wget https://go.dev/dl/go1.19.3.linux-amd64.tar.gz
sudo tar -xf go1.19.3.linux-amd64.tar.gz
sudo mv go /usr/local
```

Now, edit the following file:

``` shell
nano  ~/.profile
```

Add the following lines to the end and save with Ctrl-x:

``` shell
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Make settings effective with:

``` shell
source ~/.profile
```

Check that everything is working by running:

``` shell
go version
```
This should return something similar to:

`go version go1.19.3 linux/amd64`

#### 2. Clone and build the pastelup repo as follows:

``` shell
git clone https://github.com/pastelnetwork/pastelup.git
cd pastelup
make
```

You may need to first run:

``` shell
go mod tidy -compat=1.17
```


## Quick start guide

### Single node
#### 1. Install node

``` shell
./pastelup install node -r latest -n=mainnet --enable-service
```
Or for testnet:
``` shell
./pastelup install node -r latest -n=testnet --enable-service
```

#### 2. Start node

``` shell
./pastelup start node
```

#### 3. Update node

``` shelll
./pastelup stop node
```

```shell
./pastelup update node -r latest -n=mainnet
```
Or for testnet:
```shell
./pastelup update node -r latest -n=testnet
```

### Walletnode

> `install walletnode` will ask about whether you want to install bridge service or not. It is **RECOMMENDED** to install bridge service.

> If you opt-in for bridge install, then the first time you start walletnode with `start walletnode` command, it will guide you either to generate new address, new artist PastelID and try to register PastelID on the network or to select existing PastelID from the list of registered PastelIDs on THIS node. 
> As alternative, in case you already have a registered PastelID on this NODE, you can add it and its passphrase into the bridge config file (`bridge.yml`) so that `start` command may not ask.
> ```shell
> download:
>     connections: 10
>     connections_refresh_timeout: 300
>     passphrase: "<PASTELID>"
>     pastel_id: "<PASSPHRASE>"
> ...
> ```

#### 1. Install walletnode
   
``` shell
./pastelup install walletnode -r latest
```
Or for testnet:
``` shell
./pastelup install walletnode -r latest -n=testnet
```

#### 2. Start walletnode

``` shell
./pastelup start walletnode
```

#### 3. Update walletnode

```shell
./pastelup update walletnode -r latest
```
Or for testnet:
``` shell
./pastelup update walletnode -r latest -n=testnet
```

### Supernode

> Supernode can only be installed on Linux OS.

There are two ways to run supernode:
* **Simple** - also called **HOT** mode - when address with the collateral transaction of 5M PSL (or 1M LSP - on testnet) stored in the wallet on the host where supernode is running
* **Secure** - also called **HOT/COLD** mode - when address with the collateral transaction of 5M PSL (or 1M LSP - on testnet) stored in the wallet on the different host (cold wallet) from the host where supernode is running

It is **RECOMMENDED** to use **Secure** mode. But this guide will explain install and start for both modes.

#### A. Install supernode in **HOT** mode - on the single host

##### 1. Install supernode
``` shell
./pastelup install supernode -r latest
```
Or for testnet:
``` shell
./pastelup install supernode -r latest -n=testnet
```

##### 2. Initialize **_newly_** installed supernode

> You should only run this command after first installation of supernode.
> If you already have initialized supernode, then you can skip this step.

``` shell
 ./pastelup init supernode --new --name=<SN-name> --activate
```

Here:
- `SN-name` is the name you want to use to address your SN in the masternode.conf file. This name has meaning only on the host where supernode is running. It is not used on the network.
- `--activate` is optional. If you want to activate your SN immediately after initialization, then add this flag. Otherwise, you can activate it later with `pastel-cli masternode start-alias <SN-name>` command.

The above command will:
- ask for passphrase
- create and register new SN's PastelID
- asks for collateral transaction `txid` and `vout` index
  - if no collateral was sent yet, it will offer to create new address and will wait until collateral is sent to it and the transaction is confirmed
- create masternode.conf file and add configuration against the provided node alias --name
- start pasteld as masternode
- activate pasteld as masternode
- start rq-server, dd-server and hermes and supernode services

Alternatively, if you already know collateral transaction `txid` and `vout` index, then you can initialize supernode with the following command:
``` shell
 ./pastelup init supernode --new --name=<SN-name> --txid=<txid> --ind=<vout index> --activate
```

After initialization, you can check the status of your supernode with the following command:
``` shell
pastel-cli masternode status
```

Verify it returns `masternode successfully started` message.

##### 3. Start supernode

> You don't need to start supernode right after initialization. You only need to start it if it was stopped before.

``` shell
./pastelup start supernode
```
The above command will start following services:
- rq-server
- dd-server
- hermes (if you chose to install it during installation)
- supernode

##### 4. Update supernode

``` shell
./pastelup update supernode  --name=<SN-name> -r latest
```

#### B. Install supernode in **HOT/COLD** mode - on the two hosts

Please refer to the following guide to install supernode in **HOT/COLD** mode:
[How cold-hot config works](https://docs.pastel.network/development-guide/supernode)


## Default settings for all commands

### default_working_dir

The path depends on the OS:
* MacOS `$HOME/Library/Application Support/Pastel`
* Linux `$HOME/.pastel`
* Windows (>= Vista) `%userprofile%\AppData\Roaming\Pastel`

### default_exec_dir

The path depends on the OS:
* MacOS `$HOME/Applications/PastelWallet`
* Linux `$HOME/pastel`
* Windows (>= Vista) `%userprofile%\AppData\Roaming\PastelWallet`


## Testing 
All tests are contained in the `test/` directory. You can invoke the tests by running
```
make build-test-img
make <test-walletnode|test-local-supernode|test-local-supernode-service>
```
This will run the associated script found in `test/scripts/` inside a docker container to validate specific functionality of `pastelup`.

## More information

https://docs.pastel.network/development-guide/types-of-pastel-installations

## Additional information
### Remote operations

Some commands support remote operations (CURRENTLY ONLY `install`). It means that you can run them on the remote host. <br>
To do that, you need to specify `remote`, and `--ssh-ip`, `--ssh-user` and either `--ssh-user-pw` or `--ssh-key` options.<br>
You also in most cases must specify `--force` option to confirm any questions that may appear during the operation, as they will be asked on the remote host.<br> 
Remote ssh-user must have sudo privileges on the remote host. <br>
And if remote host is not configured to run sudo commands without password, then you must specify `--ssh-user-pw` option.

#### Remote Install
``` shell
./pastelup install node remote -r latest -n mainnet -ssh-ip <remote-host-ip> --ssh-user <remote-host-ssh-user> --ssh-key <path-to-ssh-key> --force
```

``` shell
./pastelup install walletnode remote -r latest -n mainnet --ssh-ip <remote-host-ip> --ssh-user <remote-host-ssh-user> --ssh-key <path-to-ssh-key> --force
```


### Command line options
#### Install
``` shell
./pastelup install node --help
NAME:
   Pastel-Utility install node - Install node

USAGE:
   Pastel-Utility install node command [command options] [arguments...]

COMMANDS:
   remote   Install on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Required, Pastel version to install
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --regen-rpc                 Optional, regenerate the random rpc user, password and chosen port. This will happen automatically if not defined already in your pastel.conf file (default: false)
   --network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```

``` shell
./pastelup install walletnode --help
NAME:
   Pastel-Utility install walletnode - Install Walletnode

USAGE:
   Pastel-Utility install walletnode command [command options] [arguments...]

COMMANDS:
   remote   Install on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Required, Pastel version to install
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --regen-rpc                 Optional, regenerate the random rpc user, password and chosen port. This will happen automatically if not defined already in your pastel.conf file (default: false)
   --network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```
```shell
./pastelup install supernode --help
NAME:
   Pastel-Utility install supernode - Install Supernode

USAGE:
   Pastel-Utility install supernode command [command options] [arguments...]

COMMANDS:
   remote   Install on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Required, Pastel version to install
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --regen-rpc                 Optional, regenerate the random rpc user, password and chosen port. This will happen automatically if not defined already in your pastel.conf file (default: false)
   --network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --user-pw value             Optional, password of current sudo user - so no sudo password request is prompted
   --no-cache                  Optional, runs the installation of python dependencies with caching turned off (default: false)
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false) 
```
#### Init SuperNode
##### In the HOT mode
``` shell
./pastelup init supernode --help
NAME:
   Pastel-Utility init supernode - Initialise local Supernode

USAGE:
   Pastel-Utility init supernode command [command options] [arguments...]

COMMANDS:
   coldhot  Initialise Supernode in Cold/Hot mode
   remote   Initialise remote Supernode
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value         Optional, Location of pastel node directory
   --work-dir value, -w value    Optional, location of working directory
   --ip value                    Optional, WAN address of the host
   --reindex, -r                 Optional, Start with reindex (default: false)
   --name value                  Required, name of the Masternode to create or update in the masternode.conf
   --new                         Required (if --add is not used), if specified, will create new masternode.conf with new Masternode record in it. (default: false)
   --add                         Required (if --new is not used), if specified, will add new Masternode record to the existing masternode.conf. (default: false)
   --pkey value                  Optional, Masternode private key, if omitted, new masternode private key will be created
   --txid value                  Required (only if --update or --create specified), collateral payment txid , transaction id of 5M collateral MN payment
   --ind value                   Required (only if --update or --create specified), collateral payment output index , output index in the transaction of 5M collateral MN payment
   --skip-collateral-validation  Optional (if both txid and ind specified), skip validation of collateral tx on this node (default: false)
   --noReindex                   Optional, disable any default --reindex (default: false)
   --pastelid value              Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered
   --passphrase value            Optional, passphrase to pastelid private key. If omitted, user will be asked interactively
   --port value                  Optional, Port for WAN IP address of the node , default - 9933 (19933 for Testnet) (default: 0)
   --rpc-ip value                Optional, supernode IP address. If omitted, value passed to --ip will be used
   --rpc-port value              Optional, supernode port, default - 4444 (14444 for Testnet (default: 0)
   --p2p-ip value                Optional, Kademlia IP address, if omitted, value passed to --ip will be used
   --p2p-port value              Optional, Kademlia port, default - 4445 (14445 for Testnet) (default: 0)
   --activate                    Optional, if specified, will try to enable node as Masternode (start-alias). (default: false)
   --log-level level             Set the log level. (default: "info")
   --log-file file               The log file to write to.
   --quiet, -q                   Disallows log output to stdout. (default: false)
   --help, -h                    show help (default: false)
```
##### In the HOT/COLD mode
```shell
./pastelup init supernode coldhot --help
NAME:
   Pastel-Utility init supernode coldhot - Initialise Supernode in Cold/Hot mode

USAGE:
   Pastel-Utility init supernode coldhot [command options] [arguments...]

OPTIONS:
   --dir value, -d value         Optional, Location of pastel node directory
   --work-dir value, -w value    Optional, location of working directory
   --ip value                    Optional, WAN address of the host
   --reindex, -r                 Optional, Start with reindex (default: false)
   --name value                  Required, name of the Masternode to create or update in the masternode.conf
   --new                         Required (if --add is not used), if specified, will create new masternode.conf with new Masternode record in it. (default: false)
   --add                         Required (if --new is not used), if specified, will add new Masternode record to the existing masternode.conf. (default: false)
   --pkey value                  Optional, Masternode private key, if omitted, new masternode private key will be created
   --txid value                  Required (only if --update or --create specified), collateral payment txid , transaction id of 5M collateral MN payment
   --ind value                   Required (only if --update or --create specified), collateral payment output index , output index in the transaction of 5M collateral MN payment
   --skip-collateral-validation  Optional (if both txid and ind specified), skip validation of collateral tx on this node (default: false)
   --noReindex                   Optional, disable any default --reindex (default: false)
   --pastelid value              Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered
   --passphrase value            Optional, passphrase to pastelid private key. If omitted, user will be asked interactively
   --port value                  Optional, Port for WAN IP address of the node , default - 9933 (19933 for Testnet) (default: 0)
   --rpc-ip value                Optional, supernode IP address. If omitted, value passed to --ip will be used
   --rpc-port value              Optional, supernode port, default - 4444 (14444 for Testnet (default: 0)
   --p2p-ip value                Optional, Kademlia IP address, if omitted, value passed to --ip will be used
   --p2p-port value              Optional, Kademlia port, default - 4445 (14445 for Testnet) (default: 0)
   --activate                    Optional, if specified, will try to enable node as Masternode (start-alias). (default: false)
   --ssh-ip value                Required, SSH address of the remote HOT node
   --ssh-port value              Optional, SSH port of the remote HOT node (default: 22)
   --ssh-user value              Optional, SSH user
   --ssh-key value               Optional, Path to SSH private key
   --remote-dir value            Optional, Location where of pastel node directory on the remote computer (default: $HOME/pastel)
   --remote-work-dir value       Optional, Location of working directory on the remote computer (default: $HOME/.pastel)
   --remote-home-dir value       Optional, Location of working directory on the remote computer (default: $HOME)
   --log-level level             Set the log level. (default: "info")
   --log-file file               The log file to write to.
   --quiet, -q                   Disallows log output to stdout. (default: false)
   --help, -h                    show help (default: false)
```
#### Start
```shell
./pastelup start node --help
NAME:
   Pastel-Utility start node - Start node

USAGE:
   Pastel-Utility start node command [command options] [arguments...]

COMMANDS:
   remote   Start on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location of pastel node directory
   --work-dir value, -w value  Optional, location of working directory
   --ip value                  Optional, WAN address of the host
   --reindex, -r               Optional, Start with reindex (default: false)
   --legacy                    Optional, pasteld version is < 1.1 (default: false)
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```
```shell
./pastelup start walletnode --help
NAME:
   Pastel-Utility start walletnode - Start Walletnode

USAGE:
   Pastel-Utility start walletnode command [command options] [arguments...]

COMMANDS:
   remote   Start on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location of pastel node directory
   --work-dir value, -w value  Optional, location of working directory
   --ip value                  Optional, WAN address of the host
   --reindex, -r               Optional, Start with reindex (default: false)
   --legacy                    Optional, pasteld version is < 1.1 (default: false)
   --development-mode          (default: false)
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```
```shell 
./pastelup start supernode --help
NAME:
   Pastel-Utility start supernode - Start Supernode

USAGE:
   Pastel-Utility start supernode command [command options] [arguments...]

COMMANDS:
   remote   Start on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location of pastel node directory
   --work-dir value, -w value  Optional, location of working directory
   --ip value                  Optional, WAN address of the host
   --reindex, -r               Optional, Start with reindex (default: false)
   --legacy                    Optional, pasteld version is < 1.1 (default: false)
   --name value                name of the Masternode to start
   --activate                  Optional, if specified, will try to enable node as Masternode (start-alias). (default: false)
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```

#### Stop
Stop commands for all tools are the same.
```shell
./pastelup stop node --help
NAME:
   Pastel-Utility stop supernode - Stop Supernode

USAGE:
   Pastel-Utility stop supernode command [command options] [arguments...]

COMMANDS:
   remote   Stop on Remote Host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location of pastel node directory
   --work-dir value, -w value  Optional, location of working directory
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```

#### Update
Update commands for all tools are the same.
```shell
./pastelup update node --help
NAME:
   Pastel-Utility update node - Update Node

USAGE:
   Pastel-Utility update node command [command options] [arguments...]

COMMANDS:
   remote   Update on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --archive-dir value         Optional, Location where to store archived backup before update
   --release value, -r value   Required, Pastel version to install
   --network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --skip-system-update        Optional, Skip System Update skips linux apt-update (default: false)
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --clean, -c                 Optional, Clean .pastel folder (default: false)
   --user-pw value             Optional, password of current sudo user - so no sudo password request is prompted
   --no-backup                 Optional, skip backing up configuration files before updating workspace (default: false)
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```