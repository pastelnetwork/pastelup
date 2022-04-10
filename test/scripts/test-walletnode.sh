#!/bin/sh

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

criteria="""
pastelup install walletnode -n=testnet --peers=18.191.71.196

pastelup start walletnode
<verify 3 processes are running: pasteld, walletnode, rq-service>
<waiting for some blocks to sync>

pastelup update node
<verify pasteld was updated>
pastelup update rq-service
<verify rq-service was updated>
pastelup update walletnode-service
<verify walletnode-service was updated>

pastelup start node
pastelup start rq-service
pastelup start walletnode-service
<verify 3 processes are running: pasteld, walletnode, rq-service>

pastelup update walletnode
<verify pasteld, rq-service, walletnode-service was updated>
<verify archive in ~/.paste_archive>

pastelup start walletnode
<verify 3 processes are running: pasteld, walletnode, rq-service>
"""

pasteldExecPath=~/pastel/pasteld
rqServiceExecPath=~/pastel/rq-service-linux-amd64
walletNodeServiceExecPath=~/pastel/walletnode-linux-amd64

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
pasteldLastModified=$(date -r $pasteldExecPath '+%s')
echo "pasteld last modified detected to be $pasteldLastModified"
rqServiceLastModified=$(date -r $rqServiceExecPath '+%s')
echo "rq-service last modified detected to be $rqServiceLastModified"
walletNodeLastModified=$(date -r $walletNodeServiceExecPath '+%s')
echo "walletnode last modified detected to be $walletNodeLastModified"

# verify we can update node
echo "updating node..."
pastelup update node --force --user-pw="$USR_PW" # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated $pasteldExecPath $pasteldLastModified "pasetld"

# verify we can update rq-service
echo "updating rq-service..."
pastelup update rq-service --force --user-pw="$USR_PW"  # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated $rqServiceExecPath $rqServiceLastModified "rq-service"

# verify we can update walletnode-service
echo "updating walletnode-service..." 
pastelup update walletnode-service --force --user-pw="$USR_PW"  # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated $walletNodeServiceExecPath $walletNodeLastModified "walletnode-service"

echo "starting node..."
pastelup start node
echo "starting rq-service..."
pastelup start rq-service
echo "starting walletnode-service..."
pastelup start walletnode-service

# validate expected processes are running
ensureServiceRunning "pasteld"
ensureServiceRunning "walletnode"
ensureServiceRunning "rq-service"

# re-init last modified timestamps
pasteldLastModified=$(date -r $pasteldExecPath '+%s')
echo "pasteld last modified detected to be $pasteldLastModified"
rqServiceLastModified=$(date -r $rqServiceExecPath '+%s')
echo "rq-service last modified detected to be $rqServiceLastModified"
walletNodeLastModified=$(date -r $walletNodeServiceExecPath '+%s')
echo "walletnode last modified detected to be $walletNodeLastModified"

pastelup update walletnode --force --user-pw="$USR_PW"  # pass in force to avoid having to say "yes" to stopping current running processes
ensureExecutableUpdated $pasteldExecPath $pasteldLastModified "pasetld"
ensureExecutableUpdated $rqServiceExecPath $rqServiceLastModified "rq-service"
ensureExecutableUpdated $walletNodeServiceExecPath $walletNodeLastModified "walletnode-service"

# verify archive worked
archiveCount=$(ls -a ~/.pastel_archives | grep .pastel_archive_ | wc -l)
echo "archive count is: $archiveCount"
 # should have 2 archive dirs ->
 #      1. from update node 
 #      2. from update walletnode
if [ "$archiveCount" -eq "2" ]; 
then
    printf "${GREEN}validated archive creation${NC}\n"
else
    printf "${RED}archive count less than expected ${NC}\n"
    exit 1
fi

pastelup start walletnode
ensureServiceRunning "pasteld"
ensureServiceRunning "walletnode"
ensureServiceRunning "rq-service"

echo "successsfully completed wallet node testing suite"
exit 0