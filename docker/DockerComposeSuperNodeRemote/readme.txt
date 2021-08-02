1. run 3 containers with docker-compose run
$sudo docker-compose run miner
$sudo docker-compose run remote
$sudo docker-compose run supernode

2.Once install, remote container need to input password automatically. Need to setup password "a" in remote container.
 e.g New Password:a
     Retype Password:a
3.Once install supernode in remote container, we need to add "testnet=1" in pastel.conf
   $sudo docker container ls -a
   $sudo docker exec pastel-utility-remote-container /bin/sh/c "echo 'testnet=1' >> /root/.pastel/pastel.conf"

4. Miner container will need to input account address. You need to input account address generated from remote container.
 e.g Input Account address: 
     tPcPvBzuUnhSxw6uKqj13ZzPqrorXYUL3KF
5. After that, miner will output transaction id. You need to input that transaction id into SuperNode container.
 e.g Input transaction id:
     2371b6376692f2a93d1e03dae2ce4ab8b80f1f8ddfc60ce2ae2cb8105181dd85
6. After that, supernode container will need to input Vout value.
   $ sudo docker container ls -a
   $ sudo docker exec supernode-container /bin/sh -c "/root/pastel/pastel-cli getinfo"
    e.g output:
    {
   "version": 1000029,
   "protocolversion": 170008,
   "walletversion": 60000,
   "balance": 0.00000,
   "blocks": 0,
   "timeoffset": 0,
   "connections": 0,
   "proxy": "",
   "difficulty": 1,
   "testnet": true,
   "keypoololdest": 1627682793,
   "keypoolsize": 101,
   "paytxfee": 0.00000,
   "relayfee": 0.00100,
   "errors": ""
   }
   //If connections : 0
    $ sudo docker exec supernode-container /bin/sh -c "/root/pastel/pastel-cli addnode 192.168.114.2 onetry"

   $ sudo docker exec supernode-container /bin/sh -c "/root/pastel/pastel-cli mnsync status"
   If output like below, it means it synced.
      {
      "AssetID": 4,
      "AssetName": "Governance",
      "AssetStartTime": 1627681914,
      "Attempt": 6,
      "IsBlockchainSynced": true,
      "IsMasternodeListSynced": true,
      "IsWinnersListSynced": true,
      "IsSynced": true,
      "IsFailed": false
      }

   After that,
   $ sudo docker exec supernode-container /bin/sh -c "/root/pastel/pastel-cli gettransaction 2371b6376692f2a93d1e03dae2ce4ab8b80f1f8ddfc60ce2ae2cb8105181dd85"

   It will output Vout Value like this.
  "details": [
    {
      "account": "",
      "address": "tPWnufmNCb1M6fy2Ygz75PSGUES4RXHmUhu",
      "category": "receive",
      "amount": 1000000.00000,
      "vout": 1,
      "size": 245
    }
  ],

--After see that, input vout value.
 e.g Input Vout Value:
     1
--Then, it will start supernode remote.


6. After that, it will be run automatically and start pasteld remotely in remote container.
   If it succeed, you will see below message in terminal.
   Remote::Supernode was started successfully.
   
   - you can check supernode-linux process in supernode container to confirm it started exactly.

--In case (Insufficent coin error) pasteld exec file is not incorrect one, need to copy that to miner container and get transaction ID.
   $ sudo docker container ls -a
   $ sudo docker exec miner /bin/sh -c "pkill pasteld"
   $ sudo docker cp pasteld miner:/root/pastel/
   $ sudo docker exec miner /bin/sh -c "chmod 777 /root/pastel/pasteld"
   $ sudo docker exec miner  /bin/sh -c "/root/pastel/pasteld --mine --daemon --testnet --reindex"
   $ sudo docker exec miner  /bin/sh -c "/root/pastel/pastel-cli sendmany '' '{\"tPcPvBzuUnhSxw6uKqj13ZzPqrorXYUL3KF\": 1000000}'"
