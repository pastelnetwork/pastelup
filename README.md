# Pastelup

[![PastelNetwork](https://circleci.com/gh/pastelnetwork/pastelup.svg?style=shield)](https://app.circleci.com/pipelines/github/pastelnetwork/pastelup)

`pastelup` is a utility that can install, initialize, start, update, and monitor Pastel Network `node`, `supernode` and `walletnode`.

## Build

#### 1. Install the latest version of golang (1.19 or higher)

First, remove existing versions of golang as follows:
```shell
sudo apt-get remove --auto-remove golang-go
sudo rm -rvf /usr/local/go
```

Then, download and install golang as follows:

```shell
wget https://go.dev/dl/go1.19.3.linux-amd64.tar.gz
sudo tar -xf go1.19.3.linux-amd64.tar.gz
sudo mv go /usr/local
```

Now, edit the following file:

```shell
nano  ~/.profile
```

Add the following lines to the end and save with Ctrl-x:

```shell
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Make settings effective with:

```shell
source ~/.profile
```

Check that everything is working by running:

```shell
go version
```
This should return something similar to:

`go version go1.19.3 linux/amd64`

#### 2. Clone and build the pastelup repo as follows:

```shell
git clone https://github.com/pastelnetwork/pastelup.git
cd pastelup
make
```

You may need to first run:

```shell
go mod tidy
```

## Test
All tests are contained in the `test/` directory. You can invoke the tests by running
```
make build-test-img
make <test-walletnode|test-local-supernode|test-local-supernode-service>
```
This will run the associated script found in `test/scripts/` inside a docker container to validate specific functionality of `pastelup`.

## More information

[Pastel Network Docs](https://docs.pastel.network/introduction/pastel-overview)
[Pastel Network Docs - Types of Pastel installations](https://docs.pastel.network/development-guide/types-of-pastel-installations)


## Command line options

[see here](https://github.com/pastelnetwork/pastelup/blob/master/pastelup-help.md)
