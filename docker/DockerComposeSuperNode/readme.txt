//Docker file , docker-compose.yml, pastel-utility file should be in same path.
//run docker-compose with below command. Super Node should be done with "docker-compose run" for user interactive mode.

$sudo docker-compose run miner

$sudo docker-compose run supernode

//After run "docker-compose run supernode", it will output account address like below
tPTrtD4BKx5Qr2LLQE8Wb4HWh52VGfoiuf9

//After run "docker-compose run miner", it needs to read "Input Account Adress", just input account address(e.g tPTrtD4BKx5Qr2LLQE8Wb4HWh52VGfoiuf9) you got from supernode. Then, it will output transaction id
c96d64f13beb3ff55277b03f11d0fa2445b2dda537ca1026a23b748895df83b3

//SuperNode command needs to read "Input transaction ID:", just input transaction id(e.g c96d64f13beb3ff55277b03f11d0fa2445b2dda537ca1026a23b748895df83b3) you got from the Hot node. Then, it will start super node. Once it synced and if output is like below it means success.

File created: /root/.pastel/supernode/supernode.yml
Configuring supernode was finished
Start supernode
Waiting for supernode started...
Supernode was started successfully
	