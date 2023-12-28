#### Install

```shell
./pastelup install node --help
NAME:
   Pastel-Utility install node - Install node

USAGE:
   Pastel-Utility install node command [command options] [arguments...]

COMMANDS:
   remote   Install on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --network value, -n value   Required, network type, can be - "mainnet" or "testnet"
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Optional, Historical Pastel version to install
   --force, -f                 Optional, Force to overwrite config files and re-download ZKSnark parameters (default: false)
   --regen-rpc                 Optional, regenerate the random rpc user, password and chosen port. This will happen automatically if not defined already in your pastel.conf file (default: false)
   --peers value, -p value     Optional, List of peers to add into pastel.conf file, must be in the format - "ip" or "ip:port"
   --log-level level           Set the log level. (default: "info")
   --log-file file             The log file to write to.
   --quiet, -q                 Disallows log output to stdout. (default: false)
   --help, -h                  show help (default: false)
```

```shell
./pastelup install walletnode --help
NAME:
   Pastel-Utility install walletnode - Install Walletnode

USAGE:
   Pastel-Utility install walletnode command [command options] [arguments...]

COMMANDS:
   remote   Install on Remote host
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --network value, -n value   Required, network type, can be - "mainnet" or "testnet"
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Optional, Historical Pastel version to install
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
   --network value, -n value   Required, network type, can be - "mainnet" or "testnet"
   --dir value, -d value       Optional, Location where to create pastel node directory
   --work-dir value, -w value  Optional, Location where to create working directory
   --release value, -r value   Optional, Historical Pastel version to install
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
```shell
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
   --release value, -r value   Optional, Historical Pastel version to install
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
