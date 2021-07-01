# Pastel Utility
`pastel-utility` is a utility that can install `supernode`/`walletnode` and start.

## Start node
1. Install node

``` shell
./pastel-utility install node
```

2. Start node

``` shell
./pastel-utility start node
```

## Start walletnode
1. Install walletnode

``` shell
./pastel-utility install walletnode
```

2. Start node

``` shell
./pastel-utility start walletnode
```

## Start supernode
1. Install supernode

``` shell
./pastel-utility install supernode
```

2. Start node

``` shell
./pastel-utility start supernode
```

### CLI Install node command options
`pastel-utility` supports the following CLI parameters:

##### --ipath
Specifies `node` executable directory. By default [default_exec_dir](#default_exec_dir).
##### --work-dir
Specifies `node` working directory. By default [default_working_dir](#default_working_dir)
##### --network
Specifies the network. By default `mainnet`.
##### --force
Replace all directory and files. By default `false`.
##### --peers
Add peers to connect Pastel blockchain.

### CLI Install walletnode command options
`pastel-utility` supports the following CLI parameters:

##### --ipath
Specifies `walletnode` executable directory. By default [default_exec_dir](#default_exec_dir).
##### --force
Replace all directory and files. By default `false`.

### CLI Install supernode command options
`pastel-utility` supports the following CLI parameters:

##### --ipath
Specifies `supernode` executable directory. By default [default_exec_dir](#default_exec_dir).
##### --force
Replace all directory and files. By default `false`.

### Default settings

##### default_working_dir

The path depends on the OS:
* MacOS `~/Library/Application Support/Pastel`
* Linux `~/.pastel`
* Windows (>= Vista) `C:\Users\Username\AppData\Roaming\Pastel`
* Windows (< Vista) `C:\Documents and Settings\Username\Application Data\Pastel`

##### default_exec_dir

The path depends on the OS:
* MacOS `~/Library/Application Support/Pastel-node`
* Linux `~/pastel-node`
* Windows (>= Vista) `C:\Users\Username\AppData\Roaming\Pastel-node`
* Windows (< Vista) `C:\Documents and Settings\Username\Application Data\Pastel-node`
