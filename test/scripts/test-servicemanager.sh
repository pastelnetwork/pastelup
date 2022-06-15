#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color


"""
systemctl --user start pastel-walletnode.service
systemctl --user status pastel-walletnode.service
systemctl --user enable pastel-walletnode.service
systemctl --user disable pastel-walletnode.service
systemctl --user stop pastel-walletnode.service
"""

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

ensureServiceStopped() 
{
    if pgrep "$@" > /dev/null
    then
        printf "${RED}$@ is still running. Exiting process with failure${NC}\n"
        exit 1 # exit from process
    else
        printf "${GREEN}$@ is running${NC}\n"
        return 0
    fi
}

ensureFileExists()
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


echo "installing walletnode..."
pastelup install walletnode -n=testnet --peers=18.191.71.196 -r beta

echo "starting walletnode..."
pastelup update install-service walletnode

echo "ensure walletnode process is running after service install ..."
ensureServiceRunning "walletnode"

status=$(systemctl --user status pastel-walletnode.service)
echo "systemd status --> $status"
if [[ $status == *"Loaded: loaded (/home/ubuntu/.config/systemd/user/pastel-walletnode.service, enabled)"* ]]; then
  printf "${GREEN}verified enabled state of systemd service${NC}\n"
  else
    printf "${RED}unable to verify enabled state of systemd service${RED}\n"
    exit 1
fi
if [[ $status == *"Active: active (running)"* ]]; then
  printf "${GREEN}verified active state of systemd service${NC}\n"
  else
    printf "${RED}unable to verify active state of systemd service${RED}\n"
    exit 1
fi

# check that service file was created
echo "ensure walletnode systemd config file exists..."
ensureFileExists ~/.config/systemd/user/pastel-walletnode.service

# check that we can reboot and it will start up
# @todo

# check that we can reinstall and it will be a noop
echo "reinstalling walletnode"
pastelup install walletnode -n=testnet --peers=18.191.71.196 -r beta
echo "ensure walletnode process is still running ..."
ensureServiceRunning "walletnode"


# check that we can stop the service
echo "stopping walletnode service..."
pastelup update remove-service walletnode
status=$(systemctl --user status pastel-walletnode.service)
echo "systemd status --> $status"
if [[ $status == *"Loaded: loaded (/home/ubuntu/.config/systemd/user/pastel-walletnode.service, disabled)"* ]]; then
  printf "${GREEN}verified disabled state of systemd service${NC}\n"
  else
    printf "${RED}unable to verify disabled state of systemd service${RED}\n"
    exit 1
fi
if [[ $status == *"Active: inactive (dead)"* ]]; then
  printf "${GREEN}verified active state of systemd service${NC}\n"
  else
    printf "${RED}unable to verify active state of systemd service${RED}\n"
    exit 1
fi
# ensureServiceStopped "walletnode" --> stopping the systemd service does NOT kill the process -- do we want to kill the process?
