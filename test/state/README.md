# Walletnode Recovery

When running tests that involve supernodes, you need 1M LSP to successfully start the nodes. To set this up, you can follow the steps in this guide.


# First Time
To setup your enviornment, you need to generate a wallet with a privatekey that has 1M LSP 
in it, so that you can run a supernode as part of the test suite. If you already have this, 
then all you need to do is ensure it's in you `test/state/env.json` file i.e. 
```
{
    "privKey": "<key>"
}
```

# Subsequent Times

## Troubleshooting

* If you get system error that the docker container ran out of space, you likely need to prune dangling/old containers/images: https://docs.docker.com/config/pruning/
