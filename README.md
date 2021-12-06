# Pastelup
`pastelup` is a utility that can install `supernode`/`walletnode` and start.

In order to build `pastelup`, pls install `golang` and `upx`:
```
sudo apt-get install upx
```
and then:
```
git clone https://github.com/pastelnetwork/pastel-utility.git
cd pastel-utility
make
```

## Install and Start

### Start node
1. Install node

Usage:
```
./pastelup install node --help
NAME:
   Pastelup install node - Install node

USAGE:
   Pastelup install node [command options] [arguments...]

OPTIONS:
   --dir value, -d value       Optional, Location where to create pastel node directory (default: "/home/bacnh/pastel")
   --work-dir value, -w value  Optional, Location where to create working directory (default: "/home/bacnh/.pastel")
   --network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --release value, -r value   Optional, Pastel version to install (default: "beta")
   --enable-service            Optional, start all apps automatically as systemd service (default: false)
   --user-pw value             Optional, password of current sudo user - so no sudo password request is prompted
   --help, -h                  show help (default: false)
   ```

``` shell
./pastelup install node --enable-service
```

For testnet:
``` shell
./pastelup install node -n=testnet --enable-service
```

2. Start node

``` shell
./pastelup start node
```

3. Update node

```shell
./pastelup update node
```

### Start walletnode
1. Install walletnode

``` shell
./pastelup install walletnode
```

For testnet:
``` shell
./pastelup install walletnode -n=testnet
```

2. Start walletnode

``` shell
./pastelup start walletnode
```

3. Update walletnode

```shell
./pastelup update walletnode
```

### Start supernode
1. Install supernode

``` shell
./pastelup install supernode
```

For testnet:
``` shell
./pastelup install supernode -n=testnet
```

2. Start **_new_** supernode

``` shell
./pastelup start supernode --create --activate --name=<local name for masternode.conf file>
```

The above command will:
- ask for passphrase
- create and register new SN's PastelID
- ask for collateral transaction txid and vout index
  - if no collateral were sent yet, it will offer to create new address and will wait until collateral is sent to it and transaction is confirmed
- create masternode.conf file
- start pasteld as masternode
- activate pasteld as masternode
- start rq-server, dd-server and supernode

3. Update supernode

```
./pastelup update supernode  --name=<local name for masternode.conf file>
```

### Install supernode remotely

In order to install all extra packages and set system services, `password` of current user with `sudo` access is needed via param `--ssh-user-pw`.

Below is example to create supernode with `testnet` network:
```
./pastelup install supernode remote \
  --ssh-ip <remote_ip> \
  --ssh-user=<remote username> \
  --ssh-user-pw=<remote_user_pw> \
  --ssh-key=$HOME/.ssh/id_rsa \
  -n=testnet \
  --force
```
### Update supernode remotely

#### Usage
```
NAME:
   pastelup update supernode remote - 

USAGE:
   pastelup update supernode remote [command options] [arguments...]

OPTIONS:
   --user-pw value             Optional, password of current sudo user - so no sudo password request is prompted
   --dir value, -d value       Optional, Location where to create pastel node directory (default: "/home/bacnh/pastel")
   --work-dir value, -w value  Optional, Location where to create working directory (default: "/home/bacnh/.pastel")
   --name value                Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)
   --ssh-ip value              Required, SSH address of the remote host
   --ssh-port value            Optional, SSH port of the remote host, default is 22 (default: 22)
   --ssh-user value            Optional, Username of user at remote host
   --ssh-key value             Optional, Path to SSH private key for SSH Key Authentication
   --bin value                 Required, local path to the local binary (pasteld, pastel-cli, rq-service, supernode) file  or a folder of binary to remote host
   --help, -h                  show help (default: false)
```
`pastelup update` will do folowing steps:
- Copy `pastelup` tool specified by `--utility-path' to `/tmp` folder of remote side
- Stop `supernode` serivces by `pastel-ulity stop supernode`
- Copy the file specified at `--bin` to `--dir` at remote side. If path is directory, it will copy all files inside that folder to remote side
- Start `supernode` again with masternode config `--name`

a) To update supernode bin to remote side:

```
./pastelup update supernode remote \
  --bin=$HOME/pastel/supernode-ubuntu20.04-amd64 \
  --name=<masternode name> \
  --ssh-ip=<remote ip> \
  --ssh-user=<remote user> \
  --user-pw=<pw of remote user> \
  --ssh-key=<private key path>
```
b) To update all binaries at once. Create a local folder and copy all binaries into a folder and execute below command with `--bin` points to that folder path:
```
./pastelup update supernode remote \ 
  --bin=<path fo that folder> \
  --name=<masternode name> \
  --ssh-ip=<remote ip> \
  --ssh-user=<remote user> \
  --user-pw=<pw of remote user> \
  --ssh-key=<private key path>
```
c) In case `--bin` is missing, the tool will update the latest from the download page

```
./pastelup update supernode remote  \
   --name=<masternode name> \
   --ssh-ip <remote ip> \
   --ssh-user <remote username> \
   --ssh-key=$HOME/.ssh/id_rsa \
   --user-pw=<pw of remote user>
```

### Stop supernode remotely

```
./pastelup stop supernode remote \
   --ssh-ip 10.211.55.5 \
   --ssh-user bacnh \
   --ssh-key $HOME/.ssh/id_rsa
```

### Start supernode-coldhot

How cold-hot is working: https://pastel.wiki/en/home/how-to-start-mn

Usage:
```
./pastel-utility start supernode-coldhot \
   --ssh-ip 10.211.55.5 \
   --ssh-user bacnh \
   --ssh-key=$HOME/.ssh/id_rsa 
   --name=mn01 
   --create
```

### Install command options

`pastelup install <node|walletnode|supernode> ...` supports the following common parameters:

- `--dir value, -d value       Optional, Location where to create pastel node directory (default: "$HOME/pastel")`
- `--work-dir value, -w value  Optional, Location where to create working directory (default: "$HOME/.pastel")`
- `--network value, -n value   Optional, network type, can be - "mainnet" or "testnet" (default: "mainnet")`
- `--force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)`
- `--peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"`
- `--release value, -r value   Optional, Pastel version to install (default: "beta")`
- `--enable-service            Optional, start all apps automatically as systemd service`
- `--user-pw value             Optional, password of current sudo user - so no sudo password request is prompted`
- `--help, -h                  show help (default: false)`

### Start command options

### Common options

`pastelup start <node|walletnode|supernode> ...` supports the following common parameters:

- `--ip value                  Optional, WAN address of the host`
- `--dir value, -d value       Optional, Location of pastel node directory (default: "$HOME/pastel")`
- `--work-dir value, -w value  Optional, location of working directory (default: "$HOME/.pastel")`
- `--reindex, -r               Optional, Start with reindex (default: false)`
- `--help, -h                  show help (default: false)`

### Walletnode specific options

`pastelup start walletnode ...` supports the following parameters in addition to common:

- `--development-mode          Optional, Starts walletnode service with swagger enabled (default: false)`

### Supernode specific options

`pastelup start walletnode ...` supports the following parameters in addition to common:

- `--activate                  Optional, if specified, will try to enable node as Masternode (start-alias). (default: false)`
- `--name value                Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)`
- `--pkey value                Optional, Masternode private key, if omitted, new masternode private key will be created`
- `--create                    Optional, if specified, will create Masternode record in the masternode.conf. (default: false)`
- `--update                    Optional, if specified, will update Masternode record in the masternode.conf. (default: false)`
- `--txid value                Required (only if --update or --create specified), collateral payment txid , transaction id of 5M collateral MN payment`
- `--ind value                 Required (only if --update or --create specified), collateral payment output index , output index in the transaction of 5M collateral MN payment`
- `--pastelid value            Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered`
- `--passphrase value          Required (only if --update or --create specified), passphrase to pastelid private key`
- `--port value                Optional, Port for WAN IP address of the node , default - 9933 (19933 for Testnet) (default: 0)`
- `--rpc-ip value              Optional, supernode IP address. If omitted, value passed to --ip will be used`
- `--rpc-port value            Optional, supernode port, default - 4444 (14444 for Testnet (default: 0)`
- `--p2p-ip value              Optional, Kademlia IP address, if omitted, value passed to --ip will be used`
- `--p2p-port value            Optional, Kademlia port, default - 4445 (14445 for Testnet) (default: 0)`
- `--ip value                  Optional, WAN address of the host`
- `--dir value, -d value       Optional, Location of pastel node directory (default: "/home/alexey/pastel")`
- `--work-dir value, -w value  Optional, location of working directory (default: "/home/alexey/.pastel")`
- `--reindex, -r               Optional, Start with reindex (default: false)`
- `--help, -h                  show help (default: false)`


## Stop
Command [stop](#stop) stops Pastel network services

### Stop all

stop ALL local Pastel Network services

``` shell
./pastelup stop all [options]
```

### Stop node

stop Pastel Core node

``` shell
./pastelup stop node [options]
```

### Stop walletnode

stop Pastel Network Walletnode (not UI Wallet!)

``` shell
./pastelup stop walletnode [options]
```

### Stop supernode

stop Pastel Network Supernode

``` shell
./pastelup stop supernode [options]
```

### Options

#### --d, --dir

Optional, location to Pastel executables installation, default - see platform specific in [install](#install) section

#### --w, --workdir

Optional, location to working directory, default - see platform specific in [install](#install) section

### Default settings

##### default_working_dir

The path depends on the OS:
* MacOS `$HOME/Library/Application Support/Pastel`
* Linux `$HOME/.pastel`
* Windows (>= Vista) `%userprofile%\AppData\Roaming\Pastel`

##### default_exec_dir

The path depends on the OS:
* MacOS `$HOME/Applications/PastelWallet`
* Linux `$HOME/pastel-node`
* Windows (>= Vista) `%userprofile%\AppData\Roaming\PastelWallet`