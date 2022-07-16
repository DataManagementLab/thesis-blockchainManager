# Blockchain Enabler Platform

This project is about a blockchain platform which would allow the user to create private blockchain with least hassle.

## Description 

Welcome to the Blockchain enabler Platform which makes handling and working with private blockchain easier and intuitive for the user. 
Current implementation would include [Hyperledger fabric](https://www.hyperledger.org/use/fabric) and [Ethereum](https://ethereum.org/en/)
The user can create and manage multiple blockchain networks using the platform.

Currently the user can create the blockchain network, invite other parties to this network and leave the network.
Apart from this the user can also decide where he wants to store the related data, as they can utilize either local directory or the volumes provided by docker to store and track data.
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
8. Files that are needed
9. Conclusion
10. License

## Introduction

The Blockchain Enabler Platform serves as a tool for creating and managing blockchain network.
It provides the user ease to create a network and invite other parties to join this network. 
It can be thought of as social media application wherin the parties who know each other, can form a group or separate chat to communicated among them.
Currently the Work is under progress and this is implemented currently as a CLI using the go module [cobra](https://pkg.go.dev/github.com/spf13/cobra)
The language of implementation in entirety is [golang](https://go.dev/).
It also utilizes containers to create the network using [docker](https://docs.docker.com/).
The documentation is done using the [godoc](https://go.dev/blog/godoc)

The project allows the user to create private blockchain networks using [Hyperledger fabric](https://www.hyperledger.org/use/fabric) and [Ethereum](https://ethereum.org/en/). The Hyperledger fabric network is chosen and created by default. 

The project is also inspired by the [Hyperledger Firefly](https://www.hyperledger.org/use/firefly).

## Motivation

The main motivation behind this project is to make the understanding and usability of blockchain simpler.
Thus making it usable by anyone with basic understanding of the blockchain technology.

The platform aims at providing a decentralized platform for creating and joining the network, where no central entity is entirely responsible to initiating the communication between the parties. 

The user can also deploy their [smart contract](https://www.ibm.com/topics/smart-contracts) on the network and interact with it using the nodes.


## Requirements
The project requires some dependencies to be installed before using this platform.

1. [Docker](https://docs.docker.com/get-docker/)
2. [Docker compose](https://docs.docker.com/compose/install/)
3. [Golang](https://go.dev/doc/install)

## Setting up Project

### Installations

1. Golang

Our current setup focusses on a Ubuntu 20.04 LTS Virtual machine and in order to install the golang, we have used the following approach.

```bash

wget -c https://dl.google.com/go/go1.18.4.linux-amd64.tar.gz

shasum -a 256 go1.18.4.linux-amd64.tar.gz

sudo tar -C /usr/local -xzf go1.18.4.linux-amd64.tar.gz

mkdir -p $HOME/go/{bin,src,pkg}
```

Also add the environment variables to your bash_rc
```bash
 <!-- Go environment -->
export GOROOT=/usr/local/go
<!-- Change the GOPATH value if your workspace is not $HOME/go -->
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOROOT/bin

```
After this run source ~/.bash_rc

you can check if the golang has been successfully installed into the machine via

```bash
go version
```
Use this tutorial for more [help](https://shakib37.medium.com/how-to-install-golang-f8cbe15baa7c)

2. Docker

In order to install use docker installation [tutorial](https://docs.docker.com/engine/install/ubuntu/)

Here the steps are mentioned to get the latest version of docker

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
Now install the docker engine.

```bash

sudo apt-get update

sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin
```

3. Docker compose

For our setup to work, we still need to install the docker compose.

For more information on this process, look through the documentation for [installing docker compose on Ubuntu](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-compose-on-ubuntu-20-04)


Here are the listed steps that need to be followed.

```bash

sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

sudo chmod +x /usr/local/bin/docker-compose

docker-compose --version

```

This should result in informing you about the version used. 

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

The platform is currently a command line which handles every thing from the network initialization, creation, joining to leaving.
The commands are made intuitive for the user to understand.

As mentioned the process is divided into different steps.

### First step being the initialization of the network.

This includes different tasks while initializing the network done using the **init** command 

```cli
go run main.go init ${USERNAME}
```
```bash
Flags:
  -s, --simpleSetup   Choose this to form a network without the orderer default: disabled
  -b, --blockchain    Provide the Blockchain you would like to use options are [fabric geth corda] (default "fabric")
  -n, --networkID     Provide the name for the network. (default "kinshuk_network1")
  -o, --orgName       Provide the name for the organization default value org1. (default "Org1")
  -v, --useVolume     enable or disable the use of Volume default: disabled
```


The **init** command is the most essential command, it **initializes** the network.

By Initilaizing the network, it creates the necessary files that are needed for creating the network.

It is the preparatory phase for the network, where the important files such as docker-compose, configtx, cryptogen are created along with the entire folder structure.

Once the initialization of the network is done, then the user can now proceed to the creation phase of the network.

### The Creation Phase

The initialization phase is followed by creation of the network using files that were generated and created during the initializing phase.

The user can also decide whether they want network with all the containers ( in Fabric orderer and peer) or just create a simple setup with just one container representing the peer.

A normal network is created by default (with orderer and peer) unless chosen otherwise. 

This command works differently on different blockchain. And follows different set of steps for each [Hyperledger fabric](https://www.hyperledger.org/use/fabric) and [Ethereum](https://ethereum.org/en/).

Once the network is created, the network is joined by the parent organization peers. 

Now the organization can invite the other organization to also join this network using the join phase.


create network
```cli
go run main.go create -u ${USERID} -n ${NETWORK_NAME}
```
``` bash
Flags:
  -s, --simpleSetup     Function to enable or disable the use of Basic setup default: false
  -n, --netid string    Provide the network id of the network you want to run.
  -u, --userId string   Provide the user Id for the network you want to run.
  -v, --useVolume       enable or disable the use of Volume default: disabled
 ```

### The Join Phase

  The join phase is divided into two parts

  1. Preparation
  2. Finalize 

  1. Preparation is done by the organization which has invited the other organization to join the network.
  Thus in this phase, the iniviting organization, prepares the network by making changes to the configurations such that another organization is allowed to join the network.
  ```bash
  go run main.go join -u ${USERID} -o ${JOINING_ORG} -n ${NETWORK_OF_JOINING_ORG} -t ${TARGET_NETWORK_TO_JOIN} -j ${ORGANIZAITION_WHOSE_NETWORK_TO_BE_JOINED}
  ```

  ```bash
  Flags:
  -n, --networkId1 string   The Network the organization which wants to join another network.
  -t, --networkId2 string   The Network the organization or the target network.
  -o, --orgname1 string     The organization name which wants to join the network.
  -j, --orgname2 string     The organization name whose network is to be joined.
  -u, --userId string       The User ID for the user.
  -v, --useVolume           Function to enable or disable the use of Volume default: false

  ```
  Once the preparation phase is done, the organization can now join this network.

  In order to do so, we have another phase called the finalize phase.
  Which runs on behalf, or in the organization which wants to join the network.

  2. Finalize phase is handled by organization which wants to join the network.

  In this phase, the organization joins the network and adds its peers to the network.

  The commands for this is the same only one of the flag -f (which represents the finalize ) needs to be appended to the command.
  ```bash
   go run main.go join -u ${USERID} -o ${JOINING_ORG} -n ${NETWORK_OF_JOINING_ORG} -t ${TARGET_NETWORK_TO_JOIN} -j ${ORGANIZAITION_WHOSE_NETWORK_TO_BE_JOINED} -f
  ```
   Flags:
   ```bash
   -f, --finalize            Function to tell which phase it is as join is divided into two phases preparation and finalize. It runs on behalf of the adding network.
  ```

  ### Leave Phase

  After the join is successful, an organization, which is part of a network can also decide to leave the network. 

  This is done using the leave command. However similar to the join, the leave command is also divided into two phases,

  One of the phase is done by the organization which wants to leave the network, while other is done by Organization which is still part of the network.

  As in all of these changes, the transaction needs to be endorsed by peers, thus after the majority of peers have endorsed the transaction, only then the transaction comes into effect.
  ```bash
  go run main.go leave -u ${USERID} -o ${ORGANIZATION_WHICH_WANTS_TO_LEAVE} -n ${NETWORK_THE_ORGANIZATION_BELONGS_TO} -p ${NETWORK_WHICH_IT_WANTS_TO_LEAVE}


  Flags:
  -n, --networkName string         The Network the organization which wants to leave
  -o, --orgName string             The organization name which wants to leave the channel.
  -p, --parentNetworkName string   The parent network for the organization which wants to leave
  -u, --userId string              The User ID for the user.
  -v, --useVolume                  Function to enable or disable the use of Volume default: false

  ```

  Using this command, the organization can leave the network.

## Running the Project

This process is designed to guide you through the entire process of initializing, creating , joining and leaving the network step by step with example.

This serves as a point to check for errors, if you are facing any while replicating the commands in your network.

Our example considers two **ORGANIZATIONS**, **Org1** and **Org3**

Our first example is without Volume, later we will also showcase how it is used with volume.

Org1 is **initialized** as normal setup under the userid kinshuk.

with command 

```bash
go run main.go init kinshuk

```

A network is initialized with name kinshuk_network1, and the setup does not utilize the volumes, we can change that by using the -v flag.

Also we can change the name of the network instead by providing the network name along with the -n flag.

We can also choose which blockchain we want to use by changing choosing fabric/ether with -b flag.

As some of these are done by default so we donot need to specify them everytime we initialize the network. 


Next we create the Network using the **create** command.

```bash
go run main.go create -u kinshuk -n kinshuk_network1
```

This will create and run the containers for the network. Thus once this phase has been successfully executed, the network kinshuk_network1 is live with its containers, and organization Org1 choosen by default is the member of this network.

Now since the network is created for the Org1 , we now need to create the Organization which wants to join the network.

Here we are currently utilizing the --simplesetup which means in our network we only have a single peer container.

So in order to initialize and create the Org3, we follow the same process starting with the **init** followed by **create**

```bash

go run main.go init kinshuk -o Org3 -n kinshuk_network3 -s     

```

Here we are passing the Organization name as Org3 and the network name as kinshuk_network3 while also passing the flag for simple setup

Next we need to run the create using the command below.

```bash
go run main.go create -u kinshuk -n kinshuk_network3 -s   

```

This would create the network kinshuk_network3 with Org3 and peer0 from Org3.

Once both of our organizations are ready, we can invite Org3 to join the network created by Org1, kinshuk_network1.

Currently this step is done offline, where the Org3 sends its configuration file (organization definition file generated in create phase) to Org1.

After this the Org1 uses this org definition file in order to add Org3 to the kinshuk_network1

This is taken care in the **join** Preparation phase.


```bash
go run main.go join -u kinshuk -o Org3 -n kinshuk_network3 -m kinshuk_network1 -j Org1

```

This command uses the organization definition file provided by Org3 and then uses it to add the configuration to the network kinshuk_network1.

Once this command runs successfully, Org3 is ready to join the network.

Next is the **join** Finalize phase

```bash
go run main.go join -u kinshuk -o Org3 -n kinshuk_network3 -m kinshuk_network1 -j Org1 -f

```

This command is run by the Org3 it requires also the file used by the orderer in order to join  the network and the genesis block,

once executed, the Org3 is has now joined the kinshuk_network1


Finally if the Org3 wants, it can **leave** the network using the leave command.

```bash

go run main.go leave -u kinshuk -o Org3 -n kinshuk_network3 -p kinshuk_network1
```

This is also executed in phases whose one part is run by the Org3 and the other by the Org1 which is still part of the network.

Once this command has executed successfully, Org3 would no longer be part of the network kinshuk_network1



## License
[GNU](https://choosealicense.com/licenses/agpl-3.0/#)
