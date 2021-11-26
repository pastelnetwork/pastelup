# Pastel Utility
`pastel-utility` is a utility that can install `supernode`/`walletnode` and start.

## Install and Start

### Start node
1. Install node

``` shell
./pastel-utility install node
```

For testnet:
``` shell
./pastel-utility install node -n=testnet
```

2. Start node

``` shell
./pastel-utility start node
```

### Start walletnode
1. Install walletnode

``` shell
./pastel-utility install walletnode
```

For testnet:
``` shell
./pastel-utility install walletnode -n=testnet
```

2. Start walletnode

``` shell
./pastel-utility start walletnode
```

### Start supernode
1. Install supernode

``` shell
./pastel-utility install supernode
```

For testnet:
``` shell
./pastel-utility install supernode -n=testnet
```

2. Start **_new_** supernode

``` shell
./pastel-utility start supernode --create --activate --name=<local name for masternode.conf file>
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

### Start supernode remotely

In order to install all extra packages and set system services, `password` of current user with `sudo` access is needed via param `--ssh-user-pw`.

```
./pastel-utility install supernode remote --ssh-ip <remote_ip> --ssh-dir=<path_remote_utility_folder>/ --utility-path-to-copy=<path_local_pastel-utility> --ssh-user=bacnh --ssh-user-pw=<remote_user_pw> --ssh-key=$HOME/.ssh/id_rsa 
```

### Install command options

`pastel-utility install <node|walletnode|supernode> ...` supports the following common parameters:

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

`pastel-utility start <node|walletnode|supernode> ...` supports the following common parameters:

- `--ip value                  Optional, WAN address of the host`
- `--dir value, -d value       Optional, Location of pastel node directory (default: "$HOME/pastel")`
- `--work-dir value, -w value  Optional, location of working directory (default: "$HOME/.pastel")`
- `--reindex, -r               Optional, Start with reindex (default: false)`
- `--help, -h                  show help (default: false)`

### Walletnode specific options

`pastel-utility start walletnode ...` supports the following parameters in addition to common:

- `--development-mode          Optional, Starts walletnode service with swagger enabled (default: false)`

### Supernode specific options

`pastel-utility start walletnode ...` supports the following parameters in addition to common:

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
./pastel-utility stop all [options]
```

### Stop node

stop Pastel Core node

``` shell
./pastel-utility stop node [options]
```

### Stop walletnode

stop Pastel Network Walletnode (not UI Wallet!)

``` shell
./pastel-utility stop walletnode [options]
```

### Stop supernode

stop Pastel Network Supernode

``` shell
./pastel-utility stop supernode [options]
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