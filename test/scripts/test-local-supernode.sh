#!/bin/sh

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color


criteria="""
pastelup install node -n=testnet --peers=18.191.71.196
pastelup start node
<verify 1 processes running: pasteld>
<waiting for some blocks to sync>
pastelup install supernode
<verify pasteld was updated>
<verify rq-service, dd-service and supernode was installed>
<verify dd-service working dir - pastel_dupe_detection_service - was created and not empty>

pastelup init supernode --name=<random-name> --new --activate
<this will be long and interactive process: 
<  - it has to sync the whole testnet chain>
<  - then it will ask for missing info and provide address to send 1M LSP>

<verify 5 processes are running: pasteld, supernode, rq-service, dd-service, dd-img-server>
./pastel/pastel-cli masternode status

pastelup update node
<verify pasteld was updated>
pastelup update rq-service
<verify rq-service was updated>
pastelup update dd-service
<verify dd-service was updated>
pastelup update supernode-service
<verify supernode-service was updated>

pastelup start masternode --name=<random-name>
pastelup start rq-service
pastelup start dd-service
pastelup start supernode-service
<verify 5 processes are running: pasteld, supernode, rq-service, dd-service, dd-img-server>

pastelup update supernode
<verify pasteld, rq-service, dd-service, supernode-service was updated>
<verify archive in ~/.paste_archive>

pastelup start supernode --name=<random-name>
<verify 5 processes running: pasteld, supernode, rq-service, dd-service, dd-img-server>
"""

EXRTA_INSTALL_FLAGS=$1 # used for invoker to pass additional pastelup install commands (i.e. --enable-service)

pastelCLIExec=~/pastel/pastel-cli
pasteldExec=~/pastel/pasteld
rqServiceExec=~/pastel/rq-service-linux-amd64
walletnodeExec=~/pastel/walletnode-linux-amd64
supernodeExec=~/pastel/supernode-linux-amd64
ddServiceExec=~/pastel/dd-service-linux-amd64

ddServiceFolder=~/pastel/dd-service
dupeDetectionFolder=~/pastel_dupe_detection_service


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

ensureExectutableExists()
{
    if test -f "$1";
    then
        printf "${GREEN}$1 exists${NC}\n"
        return 0
    else 
        printf "${RED}$1 does not exist${NC}\n"
        exit 1
    fi
}

ensureDirNotEmpty()
{
    if [ -z "$(ls -A $1)" ]; 
    then
        printf "${RED}directory $1 is empty${NC}\n"
        exit 0
    else
        printf "${GREEN}directory $1 exists and is not empty${NC}\n"
        return 0
    fi
}

# Start testing suite
echo "starting local supernode testing suite..."
echo "install flags are: $INSTALL_FLAGS"

echo "installing node with testnet"
pastelup install node -n=testnet --peers=18.191.71.196

echo "starting node"
pastelup start node
ensureServiceRunning "pasteld"

# validate blocks are syncing
blocks=$($pastelCLIExec getinfo | jq '.blocks')
echo "current blocks are $blocks"
echo "waiting 30s for some blocks to sync..."
sleep 30
newBlocks=$($pastelCLIExec getinfo | jq '.blocks')
echo "blocks are now $newBlocks"
if [ "$newBlocks" -gt "$blocks" ]; 
then
    echo "successfully synced new blocks";
else
    echo "failed to sync new blocks"
fi

pasteldLastModified=$(date -r $pasteldExec '+%s')
echo "pasteld last modified detected to be $pasteldLastModified"

# 
# VERIRY INSTALLATION WORKS AS EXPECTED
#
#

pastelup install supernode -n=testnet --force --user-pw="$USR_PW" $EXRTA_INSTALL_FLAGS
ensureExecutableUpdated $pasteldExec $pasteldLastModified "pasetld"
ensureExectutableExists $supernodeExec
ensureExectutableExists $rqServiceExec
ensureDirNotEmpty $ddServiceFolder
ensureDirNotEmpty $dupeDetectionFolder

# 
# RESTORE WALLET WITH 1M LSP
#   -> uses your config based on your local file at ./test/env.json at image build time
#

echo "restoring wallet with 1M LSP"
$pastelCLIExec importprivkey $(jq -r '.privKey' env.json)

trxId=$(cat env.json | jq '.trxId')
trxIndex=$(cat env.json | jq '.trxIndex')

# 
# INIT SUPERNODE
#   -> takes 1+ hrs (needs to reindex all data to build state)
# 

echo "initing supernode"
randName=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 13 ; echo '')

pastelup init supernode --name=$randName --new --activate  --txid=$trxId --ind=$trxIndex

# <verify 5 processes are running: pasteld, supernode, rq-service, dd-service, dd-img-server>
ensureServiceRunning "pasteld"
ensureServiceRunning "supernode"
ensureServiceRunning "rq-service"
ensureServiceRunning "dd-service"
ensureServiceRunning "dd-img-server"

$pastelCLIExec masternode status

# verify node updated
lastMod=$(date -r $pasteldExec '+%s')
pastelup update node
ensureExecutableUpdated $pasteldExec $lastMod "pasetld"
# verify rq-service updated
lastMod=$(date -r $rqServiceExec '+%s')
pastelup update rq-service
ensureExecutableUpdated $rqServiceExec $lastMod "rq-service"
# verify dd-service updated
lastMod=$(date -r $ddServiceExec '+%s')
pastelup update dd-service
ensureExecutableUpdated $ddServiceExec $lastMod "dd-service"
# verify supernode-service updated
lastMod=$(date -r $supernodeExec '+%s')
pastelup update supernode-service
ensureExecutableUpdated $supernodeExec $lastMod "supernode-service"

randName=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 13 ; echo '')
pastelup start masternode --name=$randName
pastelup start rq-service
pastelup start dd-service
pastelup start supernode-service

# <verify 5 processes are running: pasteld, supernode, rq-service, dd-service, dd-img-server>
ensureServiceRunning "pasteld"
ensureServiceRunning "supernode"
ensureServiceRunning "rq-service"
ensureServiceRunning "dd-service"
ensureServiceRunning "dd-img-server"


pasteldLastMod=$(date -r $pasteldExec '+%s')
rqServiceLastMod=$(date -r $rqServiceExec '+%s')
ddServiceLastMod=$(date -r $ddServiceExec '+%s')
supernodeServiceLastMod=$(date -r $supernodeExec '+%s')

pastelup update supernode
ensureExecutableUpdated $pasteldExec $pasteldLastMod "pasteld"
ensureExecutableUpdated $rqServiceExec $rqServiceLastMod "rq-service"
ensureExecutableUpdated $ddServiceExec $ddServiceLastMod "dd-service"
ensureExecutableUpdated $supernodeExec $supernodeServiceLastMod "supernode-service"

# <verify archive in ~/.paste_archive>
archiveCount=$(ls -a ~/.pastel_archives | grep .pastel_archive_ | wc -l)
if [ "$archiveCount" -eq "1" ]; 
then
    printf "${GREEN}validated archive creation${NC}\n"
else
    printf "${RED}archive count less than expected ${NC}\n"
    exit 1
fi

randName=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 13 ; echo '')
pastelup start supernode --name=<random-name>

# <verify 5 processes running: pasteld, supernode, rq-service, dd-service, dd-img-server>
ensureServiceRunning "pasteld"
ensureServiceRunning "supernode"
ensureServiceRunning "rq-service"
ensureServiceRunning "dd-service"
ensureServiceRunning "dd-img-server"

# @ TODO
#   if EXRTA_INSTALL_FLAGS == --enable-service
#       ->  <REBOOT system and verify all 5 processes are running: pasteld, supernode, rq-service, dd-service, dd-img-server>


echo "sleeping for 1h to allow for debugging"
sleep 1h


echo "successsfully completed local supernode testing suite"
exit 0