#!/bin/bash

echo "---- Starting Walletnode Recovery Process -------"
echo "this may take ~40 mintues..."
pastelup start walletnode

echo "--- reimporting private key from provided state -----"
./pastel/pastel-cli importprivkey $(jq -r '.privKey' env.json)

echo "validating wallet info"
balance=$(./pastel/pastel-cli getwalletinfo | jq -r '.balance')
echo "balance after restoration is $balance"
echo "after restoration, we need to wait for the nodes to sync and the balance to populate"
attempts=0
start_time=$SECONDS
elapsed=$(( SECONDS - start_time ))
printf "balance is now $balance - attempts:$attempts, secsElapsed:$elapsed"
while [ $balance -le 0 ]; do
   balance=$(./pastel/pastel-cli getwalletinfo | jq -r '.balance')
   attempts=$(( attempts + 1 ))
   elapsed=$(( SECONDS - start_time ))
   printf "\rbalance is now $balance - attempts:$attempts, secsElapsed:$elapsed"
   sleep 5s 
done
echo ""
elapsed=$(( SECONDS - start_time ))
echo "wallet restoration completed in $elapsed seconds"
exit 0
