#!/bin/sh

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

ensureExecutableUpdated()
{
    # $1 -> the file path of the executable
    # $2 -> the original timestamp to check against
    # $3 -> the name of the service (for logging)
    
    currLastModified=$(date -r $1 '+%s')
    if [ "$currLastModified" -gt "$2" ]; 
    then
        printf "${GREEN}$3 was updated successfully${NC}\n"
        return 0
    else
        printf "${RED}$3 was not updated! ($currLastModified < $2)${NC}\n"
        exit 1
    fi
}

echo "installing walletnode..."
pastelup install walletnode -n=testnet --peers=18.191.71.196

echo "starting walletnode..."
pastelup start walletnode

# validate expected processes are running
ensureServiceRunning "walletnode"
ensureServiceRunning "rq-service"
ensureServiceRunning "pasteld"

# validate blocks are syncing
blocks=$(~/pastel/pastel-cli getinfo | jq '.blocks')
echo "current blocks are $blocks"
echo "waiting 30s for some blocks to sync..."
sleep 30
newBlocks=$(~/pastel/pastel-cli getinfo | jq '.blocks')
echo "blocks are now $newBlocks"
if [ "$newBlocks" -gt "$blocks" ]; 
then
    echo "successfully synced new blocks";
else
    echo "failed to sync new blocks"
fi

# get file timestamps to compare to updated files for validation
pasteldLastModified=$(date -r ~/pastel/pasteld '+%s')
echo "pasteld last modified detected to be $pasteldLastModified"
rqServiceLastModified=$(date -r ~/pastel/rq-service-linux-amd64 '+%s')
echo "rq-service last modified detected to be $rqServiceLastModified"
walletNodeLastModified=$(date -r ~/pastel/walletnode-linux-amd64 '+%s')
echo "walletnode last modified detected to be $walletNodeLastModified"

# verify we can update node
echo "updating node..."
pastelup update node --force # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated "~/pastel/pasteld" $pasteldLastModified "pasetld"

# verify we can update rq-service
echo "updating rq-service..."
pastelup update node --force # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated "~/pastel/rq-service-linux-amd64" $rqServiceLastModified "rq-service"

# verify we can update walletnode-service
echo "updating walletnode-service..." 
pastelup update walletnode-service --force # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated "~/pastel/walletnode-linux-amd64" $walletNodeLastModified "walletnode-service"


sleep 1h
