# Blockchain Enabler Platform

Blockchain Enabler platform is a command line tool which allows users to create and manage Permissioned Blockchain Network in a distributed manner. It provides interface for interacting with the blockchain frameworks and also enables the user to join another Blockchain Network. 

Current implementation provides support for [Hyperledger fabric](https://www.hyperledger.org/use/fabric). 

## Table of Contents

1. Introduction to Project
2. Motivation behind Project
3. Requirements for the project
4. Setting Up Project 
  4.1 Installations
  4.2 Things to consider
5. How to use the Project
6. Running the project
7. Infrastructure of the Project
8. Conclusion
9. License

## Introduction

The Blockchain Enabler Platform serves as a tool for creating and managing blockchain network.
It provides the user ease to create a network and invite other parties to join this network. 
It is made for enterprises and organizations which wish to collaborate with other organizations over a secure channel. The organizations can form a group maintain different channel for communication with each other.

<!-- The project utilizes go module [cobra](https://pkg.go.dev/github.com/spf13/cobra)
The language of implementation in entirety is [golang](https://go.dev/).
It also utilizes containers to create the network using [docker](https://docs.docker.com/).
The documentation is done using the [godoc](https://go.dev/blog/godoc) -->

The project allows the user to create private blockchain networks using [Hyperledger fabric](https://www.hyperledger.org/use/fabric. The Hyperledger fabric network is chosen and created by default. However the platform is extendable to other blockchain frameworks as [Ethereum](https://ethereum.org/en/) and others.

The project is takes the inspiration from [Hyperledger Firefly Cli](https://github.com/hyperledger/firefly-cli).

## Motivation

The key motivation behind this project is to make make the interaction with blockchain simpler by defining a interface which is consistent accross all blockchain frameworks.

The platform aims at providing a decentralized platform for creating and joining the network, where no central entity is entirely responsible to initiating the communication between the parties. 
<!-- 
The user can also deploy their [smart contract](https://www.ibm.com/topics/smart-contracts) on the network and interact with it using the nodes. -->


## Requirements
The project requires some dependencies to be installed before using this platform.

1. [Docker](https://docs.docker.com/get-docker/)
2. [Docker compose](https://docs.docker.com/compose/install/)
3. [Golang](https://go.dev/doc/install)

## Setting up Project

The setup considers Ubuntu 20.04 LTS Virtual Machine and the software installation guides are based on it.

### Installations

1. Golang

* To install golang in your system, refer to this [page](https://go.dev/dl/)

* To install on a linux machine follow the setup below.

```bash

wget -c https://dl.google.com/go/go1.18.4.linux-amd64.tar.gz

shasum -a 256 go1.18.4.linux-amd64.tar.gz

sudo tar -C /usr/local -xzf go1.18.4.linux-amd64.tar.gz

mkdir -p $HOME/go/{bin,src,pkg}
```

* Also add the environment variables to your bash_rc
```bash
 <!-- Go environment -->
export GOROOT=/usr/local/go
<!-- Change the GOPATH value if your workspace is not $HOME/go -->
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOROOT/bin

```
* After updating the environment variable, run  
```bash 
~/.bash_rc
```
* To check if golang is successfully installed in the system, use the below command.
```bash
go version
```
* Use this tutorial for more [help](https://shakib37.medium.com/how-to-install-golang-f8cbe15baa7c)

2. Docker

* To install docker, use docker installation [tutorial](https://docs.docker.com/engine/install/ubuntu/)

* Refer to the documentation below for installing docker on linux machine
```bash

sudo apt-get update

sudo apt-get install \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

sudo mkdir -p /etc/apt/keyrings

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

```
* You need to install the docker engine, to do that refer below. 

```bash

sudo apt-get update

sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin
```

3. Docker compose

* Docker compose needs to be installed for the setup to work, to do so, 
For more information on this process, look through the documentation for [installing docker compose on Ubuntu](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-compose-on-ubuntu-20-04)


* Follow the steps below for installing it on the machine.  
```bash

sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

sudo chmod +x /usr/local/bin/docker-compose


```
* To check for successful installation of docker compose, use command below.

```bash
docker compose --version
```

### Things to consider while running this Project

Note: An important information to consider while trying to run this project is that, the docker in itself does not have the sudo permissions.

Thus it is not allowed to do its tasks, so we need to create another group wherein we provide docker with the right permission allowing docker to run on the machine.

In order to do so, I can recommend an article which would [help](https://adamtheautomator.com/docker-permission-denied/)

If not done will provide us an error that docker does not have enough priviledges.

This can be checked by running the 

```bash
docker run hello-world
```

This command should create a container print the contents and then exit the container.
However if it gives error regarding the priviledges, that means docker does not have the appropriate priviledges to function.

We can do the steps below so the error doesnot occur again.

```bash

sudo groupadd docker

sudo usermod -aG docker ${USERID}

# sudo usermod -a -G sudo ${USERID}

sudo newgrp docker 

```
You might need to restart the system after this.

## How to use the project

* Once you have all the prerequisites installed in the system, you are good to go. 

* To use the system, **clone** the project and open it in a IDE locally.

* Navigate to the project location and use the command below with different cli options.
```bash

go run main.go
```

### Intro to the command line ###

<!-- Give an idea or a view of the commands that are present a brief overview of each of them -->

The Command line supports different commands and options:


**init** This is used to initialize the network and create the organization for the network. It is the preparatory phase for the network and generates the necessary configurations for the network and organization.

**create** This command is used after the init to create the network using files that were generated and created during the initializing phase.

**add** Add command is used to add another organization to the network.

**join** Join command is used to join another network.

**leave** Leave command is used by the organization when it wants to leave the network.

**delete** Delete command removes the resources used by the network and deletes the network and the organization.

**sign** Sign command is used to sign the transaction by the members of the network to reach consensus. 


## Setting up the project ##

The setup can be done locally or remotely depending on whether the user wants the network to be joined by another organization remotely.

### Local Setup ###

Local setup helps create a network locally.

* Project uses Docker Compose for managing and creating the network. 

* To setup locally, use --local flag while initializing the network 

```cli
go run main.go init ${USERNAME} --local
```

create network
```cli
go run main.go create -u ${USERID} -n ${NETWORK_NAME}
```
* In local setup, the user can create other organizations and networks with different user id and invite them to join its network.

### Remote Setup ###

* Docker swarm is used to maintain a remote setup. 

* To create a swarm cluster with other machines that want to join network refer to [article] (https://docs.oracle.com/cd/E37670_01/E75728/html/docker-swarm-setup-init.html)

* Once cluster is created and other machines have joined the cluster, user can invite these machines to join the network.

* Creation process of network is similar to the local setup only the --local flag is not used.


```cli
go run main.go init ${USERNAME}
```

create network
```cli
go run main.go create -u ${USERID} -n ${NETWORK_NAME}
```

## Setting a two organization network on local ## 

* This is a guide for setting up a network locally and enabling other organization to join this network.

* You can find this example inside test2OrgSetupLocal under test in the repository.

* Example consider two users with name **CompanyA** and **CompanyB**

--- init ---

* Now we **initialize** both **CompanyA** and **CompanyB** using command below.

```bash
go run main.go init CompanyA

go run main.go init CompanyB

```

* This initializes network with names CompanyA_network1, CompanyB_network1 and organization names CompanyAOrg1 and CompanyBOrg1  .

* The user can also customise the network and organization names using the flags in the **init** cli.

* Once the network is successfully initialized, we can move forward to create the network and the organization.

* Init also supports a simple setup with only the Organization being created instead of the entire network. Using the -s flag. 
* With the simple setup, create is not required and the containers are created directly in the **join** phase.

--- create ---

```bash
go run main.go create -u CompanyA 

go run main.go create -u CompanyB

```

* This creates the containers for the initialized networks and organization.

* Now both the Networks and Organizations with the user CompanyA and Company are created.

* So far we have CompanyA_network1 created by the organization CompanyA_Organization and CompanyB_network1 created by the organization CompanyB_Organization.

* These two networks and organizations function independently of each other. 

--- add ---

* To fully realize the benefits of collaboration, we need to make things bit more complicated. Not really :)

* Now we consider the case where CompanyBOrg1 wishes to collaborate with CompanyAOrg1 by joining its network CompanyA_Network1.

* For this CompanyBOrg1 needs to send the invite file to the CompanyAOrg1 which can be found inside the 
filepath as ~$HOME/.enabler/platform/{userid}/${network_name}/enabler/ {$OrganizationName}_invite.zip

* We assume formal communication between CompanyA and CompanyB for tranferring the zip files.

* Once CompanyAOrg1 receives the file from the CompanyBOrg1, then it can add CompanyBOrg1 to its network.

* This command adds the configuration details for CompanyB to the CompanyA_network.

* This file needs to be signed by the CompanyAOrg1 and sent to other participants of the network and finally added to the network. 

```bash
go run main.go add -u CompanyA -z /path_to_/CompanyBOrg1_invite.zip

```

--- join ---

* After the CompanyAOrg1 has successfully added CompanyBOrg1 to the network, the CompanyBOrg1 can now join the network. 

* For this CompanyBOrg1 requires the acceptance file from CompanyAOrg1 which contains information about the network. This zip file is generated in the create phase for CompanyA with name CompanyAOrg1_accept_transfer.zip.

* CompanyAOrg1 needs to send this file to the CompanyBOrg1 using any formal mode of communication.

* To join the network, now the CompanyB needs to run the join while passing the zip file received from the companyA.

```bash
go run main.go join -u CompanyB -z /path_to_/CompanyAOrg1_accept_transfer.zip

```
* With this the CompanyBOrg1 is also part of the CompanyA_network1 and can collaborate with CompanyAOrg1 on it.

![TwoOrganizationSetup](https://i.ibb.co/kHcKMky/2-Org-Seq-Diagraminit.jpg)

This whole setup operation is reflected using the sequence diagram.

In order to verify that both Organizations are part of the same network, we can go to the peer and execute the command.  

```bash
peer channel list
```

This will show that both organizations have joined the channel for CompanyA.

<!-- 

```bash

go run main.go leave -u CompanyB -o CompanyAOrg1 -n CompanyA_network1 
```


where -o and -n specify the network CompanyB operated Organization wants to leave, in our case it is CompanyAOrg1 and CompanyA_network1, 
once this is run, now the CompanyBOrg1 is no longer part of the network CompanyA_network1 and would not receive any new updates. -->


## Setting a three organization network on local ## 

* The setup of 3 Organization is similar to the 2 Organization setup as seen earlier.

* After we have 2 Organizations on a network, and we wish to collaborate with another Organization CompanyC_Org1 on the same network, we can invite CompanyC to join the network.

* First we need to create the CompanyC organization.

--- init ---

* Initialize the CompanyC using the simple setup so only the Organization is created without its own network.

* Use command below for simple setup.

```bash
go run main.go init companyC -s --local

```

* In simple setup we donot need to run the **create** for instantiating the containers, they are done using  the **join** before joining another network.

* Next send the zip invite file from the CompanyC_Org1 to the CompanyA_Org1 using any kind of tranfer.

--- add ---

* After the file has been received by ComapnyA_Org1, it needs to be signed by it and by a majority of participants of the network.

* This is done using the  **add** command.

```bash
go run main.go add -u CompanyA -z path_to/CompanyCOrg1_Invite.zip
```
--- sign ---

* Next since CompanyB_Org1 is also part of the network, it also needs to endorse this and then upload it to the network.

* For this CompanyA_Org1 needs to send the CompanyCOrg1_sign_transfer.zip file which is generated by the CompanyA_Org1 to CompanyB_Org1 using any form of formal transfer mechanism.

```bash
go run main.go sign -u CompanyB --update -z path_to/CompanyCOrg1_sign_transfer.zip
```

* The **update** flag is necessary for updating the configuration onto the network.

* The **update** flag needs to be used only when a majority of endorsements have been reached.

* After the sign is successfully done, the network if ready for the CompanyC to join it.

--- join ---

* Before doing that CompanyA needs to send the CompanyAOrg1_accept_transfer file to the CompanyC which contains the invite for the CompanyC.

* Once the CompanyC has formally received this file, then it can join the network using the join command.

```bash
go run main.go join -u CompanyC -z /path_to_/CompanyAOrg1_accept_transfer.zip

```

![ThreeOrganizationSetup](https://i.ibb.co/PQNv11T/3-Org-Setup.png)

This whole setup operation is reflected using the sequence diagram.


In order to verify that all Organizations are part of the same network, we can go to the peer of each of them  and execute the command.  

```bash
peer channel list
```


This will list all the channels joined by them and each of the organization will also list the one created by the CompanyA thus verifying that all of them are part of the same network.


## Additional Information on the Commands ##
### First step being the initialization of the network.

This includes different tasks while initializing the network done using the **init** command 

```cli
go run main.go init ${USERNAME}
```

The **init** command is the most essential command for forming the network, it **initializes** the network.

It needs to be initialized for both the organization as well as the network.

#NOTE Init also provides a --simpleSetup flag which creates only the Organization and its components. 

Once the initialization of the network is done, then the user can now proceed to the creation phase of the network.

### The Creation Phase

The initialization phase is followed by creation of the network using files that were generated using the init.

The main idea of this command is to instantiate the containers, for the newtwork so that they can communicate with each other.  

This command works differently on different blockchain and follows different set of steps for each [Hyperledger fabric](https://www.hyperledger.org/use/fabric) and [Ethereum](https://ethereum.org/en/).

Once the network is created, the network is fully functional network with multiple components.  

Now the organization can invite the other organization to also join this network using the join phase.

create network
```cli
go run main.go create -u ${USERID} -n ${NETWORK_NAME}
```
### Joining another Network ###

Joining another network is divided into few smaller commandlines,

**add** Needs to be run by the Organization whose network is to be joined.
**join** Needs to be run by the Organization which joins the network.
**sign** Needs to be run by the organizations part of the network.

#### The ADD Phase ####

  In the Add phase, the organization can add other organization to join its network, for this it also requires a invite zip file from the organization which wants to join its network.

  ```bash
  go run main.go add -u ${USERID_Inviter Organization} -z ${Zip file provided by the other organization}
  ```

  Once the Invite phase is done, the organization can now join this network.

  In order to do so, we have another phase called the Join phase.
  Which is to be ran in the organization which wants to join the network.
  
#### The Join Phase ####  

  In this phase, the organization joins the network and adds its peers to the network.

  
```bash
  go run main.go join -u ${USERID_Invitee Organization} -z ${Zip file provided by the other inviter organization}
  ```

  In join we pass the zip file from the inviter Organization denoted with _accept inside the network folder structure to the invitee organization

#### Sign Phase ####

This command needs to be explicitly called when there are multiple Organizations in a network, So in order to maintain a consensus between the participants, before taking any steps or adding any other organization, all or the majority of the participants in the network should agree. And sign command is used to endorse the transaction by a given organization in this case. 

```bash
go run main.go sign -u ${Organization Part of Network} -z ${Zip file provided for endorsement or signing}.
```
Once an organization signs this, this organization also needs to send it to other Organization part of the network, and that organization to the other. 

The sign command also has other flag --update which is used when the majority of endorsements have been reached and the signed transaction can be directly uploaded to the network. After the --update, the network is ready for the endorsed transaction. 

  ### Leave Phase

  After the join is successful, an organization, which is part of a network can also decide to leave the network. 

  This is done using the leave command. However similar to the join, the leave command is also divided into two phases,

  One of the phase is done by the organization which wants to leave the network, while other is done by Organization which is still part of the network.

  As in all of these changes, the transaction needs to be endorsed by peers, thus after the majority of peers have endorsed the transaction, only then the transaction comes into effect.
  
  Using this command, the organization can leave the network.

## License
[GNU](https://choosealicense.com/licenses/agpl-3.0/#)
