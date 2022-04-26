#!/bin/bash

echo "polling balance... if you just improrted your private key, it may take ~1 hour to realize"
balance=$(~/pastel/pastel-cli getwalletinfo | jq -r '.balance')
connections=$(~/pastel/pastel-cli getinfo | jq -r '.connections')
echo "balance is $balance and we have $connections connections"
attempts=0
start_time=$SECONDS
elapsed=$(( (SECONDS - start_time) / 60 ))
printf "balance is now $balance - attempts: $attempts, elapsed: $elapsed mins, connections: $connections"
while [ $balance -le 0 ]; do
   balance=$(~/pastel/pastel-cli getwalletinfo | jq -r '.balance')
   connections=$(~/pastel/pastel-cli getinfo | jq -r '.connections')
   attempts=$(( attempts + 1 ))
   elapsed=$(( (SECONDS - start_time) / 60 ))
   printf "\rbalance is now $balance - attempts: $attempts, elapsed: $elapsed mins, connections: $connections"
   sleep 5s 
done
echo ""
elapsed=$(( (SECONDS - start_time) / 60  ))
echo "wallet restoration completed in $elapsed mins"
exit 0
