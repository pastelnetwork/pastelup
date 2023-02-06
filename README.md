# Pastelup

[![PastelNetwork](https://circleci.com/gh/pastelnetwork/pastelup.svg?style=shield)](https://app.circleci.com/pipelines/github/pastelnetwork/pastelup)

`pastelup` is a utility that can install, initialize, start, updatr and monitor Pastel Network `node`, supernode` and `walletnode`.

## Build

#### 1. Install the latest version of golang (1.17 or higher)

First, remove existing versions of golang as follows:
```
sudo apt-get remove --auto-remove golang-go
sudo rm -rvf /usr/local/go
```

Then, download and install golang as follows:

```
wget https://go.dev/dl/go1.19.3.linux-amd64.tar.gz
sudo tar -xf go1.19.3.linux-amd64.tar.gz
sudo mv go /usr/local
```

Now, edit the following file:

```
nano  ~/.profile
```

Add the following lines to the end and save with Ctrl-x:

```
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Make settings effective with:

```
source ~/.profile
```

Check that everything is working by running:

```
go version
```
This should return something similar to:

`go version go1.19.3 linux/amd64`

#### 2. Clone and build the pastelup repo as follows:

```
git clone https://github.com/pastelnetwork/pastelup.git
cd pastelup
make
```

You may need to first run:

```
go mod tidy -compat=1.17
```


## Quick start guide

### Start single node
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

### Start walletnode

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

### Start supernode

> Supernode can only be installed on Linux OS.

There are two ways to run supernode:
* **Simple** - also called **HOT/HOT** mode - when address with the collateral transaction of 5M PSL (or 1M LSP - on testnet) stored in the wallet on the host where supernode is running
* **Secure** - also called **HOT/COLD** mode - when address with the collateral transaction of 5M PSL (or 1M LSP - on testnet) stored in the wallet on the different host (cold wallet) from the host where supernode is running

It is **RECOMMENDED** to use **Secure** mode. But this guide will explain install and start for both modes.

#### A. Install supernode in **COLD/COLD** mode - on the single host

##### 1. Install supernode
``` shell
./pastelup install supernode -r latest
```
Or for testnet:
``` shell
./pastelup install supernode -r latest -n=testnet
```

##### 2. Initialize **_newly_** installed supernode

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
The above command will:
- start rq-server, dd-server and hermes and supernode services

##### 4. Update supernode

``` shell
./pastelup update supernode  --name=<SN-name> -r latest
```

#### B. Install supernode in **HOT/COLD** mode - on the two hosts

Pelase refer to the following guide to install supernode in **HOT/COLD** mode:
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