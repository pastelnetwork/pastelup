#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color


ensureServiceRunning() 
{
    if pgrep "$@" > /dev/null
    then
        printf "${GREEN}$@ is running${NC}\n"
        return 0
    else
        printf "${RED}$@ is not running. Exiting process with failure${NC}\n"
        exit 1 # exit from process
    fi
}

echo "installing walletnode..."
pastelup install walletnode -n=testnet --peers=18.191.71.196 -r beta

echo "starting walletnode..."
pastelup update install-service walletnode

# check that service file was created
ls ~/.config/systemd/user

# check that we can reinstall and it will be a noop

# check that we can stop the service


