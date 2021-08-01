1.Docker file , docker-compose.yml, pastel-utility file should be in same path.
//run docker-compose with below command. Super Node should be done with "docker-compose run" for user interactive mode.

$sudo docker-compose run miner

$sudo docker-compose run supernode

2.After run "docker-compose run supernode", it will output account address like below
tPTrtD4BKx5Qr2LLQE8Wb4HWh52VGfoiuf9

3.After run "docker-compose run miner", it needs to read "Input Account Adress", just input account address you got from supernode. Then, it will output transaction id
c96d64f13beb3ff55277b03f11d0fa2445b2dda537ca1026a23b748895df83b3

4. SuperNode command needs to read "Input transaction ID:", just input transaction id you got from miner.
 e.g Input transaction ID:
    c96d64f13beb3ff55277b03f11d0fa2445b2dda537ca1026a23b748895df83b3
5.After that, miner container will need to input Vout Value.
   $ sudo docker container ls -a
   $ sudo docker exec supernode-container /bin/sh -c "/root/pastel/pastel-cli getinfo"
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
// Then, it will start super node. Once it synced and if output is like below it means success.

File created: /root/.pastel/supernode/supernode.yml
Configuring supernode was finished
Start supernode
Waiting for supernode started...
Supernode was started successfully
	

//If you get "Insufficient coin" error, you need to copy exact pasteld file into miner container.
--In case (Insufficent coin error) pasteld exec file is not incorrect one, need to copy that to miner container.
   $ sudo docker container ls -a
   $ sudo docker exec miner /bin/sh -c "pkill pasteld"
   $ sudo docker cp pasteld miner:/root/pastel/
   $ sudo docker exec miner /bin/sh -c "chmod 777 /root/pastel/pasteld"
   $ sudo docker exec miner  /bin/sh -c "/root/pastel/pasteld --mine --daemon --testnet --reindex"
   $ sudo docker exec miner  /bin/sh -c "/root/pastel/pastel-cli sendmany '' '{\"tPcPvBzuUnhSxw6uKqj13ZzPqrorXYUL3KF\":    1000000}'"

