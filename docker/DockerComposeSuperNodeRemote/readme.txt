1. run 3 containers with docker-compose run
$sudo docker-compose run miner
$sudo docker-compose run remote
$sudo docker-compose run supernode

2. Miner container will need to input account address. You need to input account address generated from remote container.
 e.g Input Account address: 
     tPcPvBzuUnhSxw6uKqj13ZzPqrorXYUL3KF
3. After that, miner will output transaction id. You need to input that transaction id into SuperNode container.
 e.g Input transaction id:
     2371b6376692f2a93d1e03dae2ce4ab8b80f1f8ddfc60ce2ae2cb8105181dd85
4. After that, it will be run automatically and start pasteld remotely in remote container.
   If it succeed, you will see below message in terminal.
   Remote::Supernode was started successfully.
   
   - you can check supernode-linux process in supernode container to confirm it started exactly.

--In case pasteld exec file is not incorrect one, need to copy that to miner container.
