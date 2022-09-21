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
  -b, --blockchain string   Provide the Blockchain you would like to use options are [fabric geth corda] (default "fabric")
  -n, --networkID string    Provide the name for the network. The default value is ${USERNAME}_network1
  -o, --orgName string      Provide the name for the organization default value ${USERNAME}Org1.
```

The **init** command is the most essential command for forming the network, it **initializes** the network.

In the init , an important option to choose from is the -s flag which initializes network as simple setup with just one peer in the organization and without any other components or containers. 

#NOTE in case of simple setup , we donot need to call the create command , instead the network is created directly in the accept phase.

The option is to have the network with other components for the Organization apart from the peer. 

By Initilaizing the network, it is the preparatory phase for the creating network which generates necessary files needed for the network.

It stores these files under the right folder structure.

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
``` bash
Flags:
  -n, --netid string    Provide the network id of the network you want to run, this network id is needed if you intend of creating a different network than the one initialized.
  -u, --userId string   Provide the user Id which wants to create the network, it should be same as one passed in init phase. 
 ```

### The Join Phase

  The join phase is divided into two parts

  1. Invite
  2. Accept 

  1. Invite is done by the organization which has invited the other organization to join the network.
  Thus in this phase, the iniviting organization, prepares the network by making changes to the configurations such that another organization is allowed to join the network.
  ```bash
  go run main.go invite -u ${USERID_Inviter Organization} -z ${Zip file provided by the other organization}
  ```

  ```bash
  Flags:
  -u, --userId string       The User ID for the user.
  -z  --zipFile string      Zip file containing information of the joining organization. 
  -v, --useVolume           Function to enable or disable the use of Volume default: false

  ```
  Once the Invite phase is done, the organization can now join this network.

  In order to do so, we have another phase called the Accept phase.
  Which runs on behalf, or in the organization which wants to join the network.
  
  The zip file is generated inside the folder for the network and it needs to be passed to the inviter only then the invitee can join the network.

  2. Accept phase is handled by organization which wants to join the network.

  In this phase, the organization joins the network and adds its peers to the network.

  
```bash
  go run main.go invite -u ${USERID_Inviter Organization} -z ${Zip file provided by the other invitee organization}
  ```

  ```bash
  Flags:
  -u, --userId string       The User ID for the user.
  -z  --zipFile string      Zip file containing information of the joining organization. 
  -v, --useVolume           Function to enable or disable the use of Volume default: false

  ```
The accept phase looks similar to the invite in the command line however, we pass different zip files in each of these phases.

In accept we pass the zip file from the inviter Organization denoted with _accept inside the network folder structure to the invitee organization

  ### Leave Phase

  After the join is successful, an organization, which is part of a network can also decide to leave the network. 

  This is done using the leave command. However similar to the join, the leave command is also divided into two phases,

  One of the phase is done by the organization which wants to leave the network, while other is done by Organization which is still part of the network.

  As in all of these changes, the transaction needs to be endorsed by peers, thus after the majority of peers have endorsed the transaction, only then the transaction comes into effect.
  ```bash
  go run main.go leave -u ${USERID} -o ${ORGANIZATION_WHICH_WANTS_TO_LEAVE} -n ${NETWORK_THE_ORGANIZATION_BELONGS_TO}

  Flags:
  -n, --networkName string         The Network the organization which wants to leave
  -o, --orgName string             The organization name which wants to leave the channel.
  -u, --userId string              The User ID for the user.
  -v, --useVolume                  Function to enable or disable the use of Volume default: false

  ```

  Using this command, the organization can leave the network.

## Running the Project

This process is designed to guide you through the entire process of initializing, creating , joining and leaving the network step by step with example.

This serves as a point to check for errors, if you are facing any while replicating the commands in your network.

Our example considers two users, **CompanyA** and **CompanyB**

Our first example is without Volume, later we will also showcase how it is used with volume.

CompanyA **initializes** a simple network with userid as CompanyA.

with command 

```bash
go run main.go init CompanyA

```

A network is initialized with name CompanyA_network1. This network has default organization CompanyAOrg1. However this organization name could also be changed in the init command using -o flag. 

We can also choose which blockchain we want to use by changing choosing fabric/ether with -b flag.

Once this command runs, it initializes the network and creates user with id CompanyA, and under this userid , network CompanyA_network1 containing organization CompanyAOrg1 is initialized. 

Next we create the Network using the **create** command.

```bash
go run main.go create -u CompanyA 
```

This will create and run the containers for the currently initialized network. Once this phase has been successfully executed, the network CompanyA_network1 is live with its containers, and organization CompanyAOrg1 is part of this network.

So far we have seen how we can create a network using our platform.

To fully realize the benefits of collaboration, we need to make things bit more complicated. Not really :)

Lets introduce another network into this picture, by introducing this another network , we aim to join the network created by ComanyA ,CompanyA_network1

and work on the network.

To do so we follow the same step that we took for initialization of CompanyA , however this time we do it for user id CompanyB.

Also for demonstration, lets try out the -s flag for the simple setup while using init for CompanyB.

So in order to initialize , we follow the same process as for CompanyA starting with the **init** however this time we also pass flag -s for a simple setup.

```bash

go run main.go init CompanyB -s     

```

Now this will create the user with user ID CompanyB, and inside the CompanyB, would initialize a network CompanyB_network with only component CompanyBOrg1

However this network is not up and running now. Our aim with -s (simple setup) flag is that we donot have to carry on with the create phase again, and the containers are instantiated when they want to join another network. 

So this intuitively means we donot have to run the create phase for the setup. 

Also in the init phase, a zip file is generated which is used for sending it to the inviter organization.

We can located this file in the directory where the network is present. ~$HOME/.enabler/platform/{userid}/${network_name}/enabler/ {$OrganizationName}_invite.zip

THe user can find the path for this file in the command line when they run init command. 

So next step would be to located this file for the user id CompanyB and network id CompanyB_network and then send this file to the inviter which is our CompanyA.

Currently we assume any form of connection for transfering the zip file, eg via email, or file transfer, this process is handled outside of our platform. 

Once this file is located and passed to CompanyA,

Next we can begin with the join phase by starting with the invite phase. 

Remember this phase is run on the Organization which wants to invite another organization to join its network, in our case CompanyA which is inviting CompanyB to join the network CompanyA_network.


```bash
go run main.go invite -u CompanyA -z /path_to_/CompanyBOrg1_invite.zip

```

Here we pass the path to the copied invite file from CompanyB.


This command adds the configuration details for CompanyB to the CompanyA_network.

Once this command runs successfully, CompanyBOrg1 is ready to join the network.

But for this we need now a file from the CompanyA (inviter). This zip file is generated in the create phase for CompanyA with name CompanyAOrg1_accept_transfer.zip and is sent to organization, which want to join this network.

Again this is handled independent of the platform. Once this accept file has been transferred to the CompanyB, we can now run the accept phase.

To do this , we run the accept command in the host machine with user CompanyB
```bash
go run main.go accept -u CompanyB -z /path_to_/CompanyAOrg1_accept_transfer.zip

```

once executed successfully, the Organization CompanyBOrg1 is first created and then it joins the network for CompanyA , CompanyA_network


Finally once this is done, if any Organization wants, then it can also **leave** the network using the leave command.

```bash

go run main.go leave -u CompanyB -o CompanyAOrg1 -n CompanyA_network1 
```


where -o and -n specify the network CompanyB operated Organization wants to leave, in our case it is CompanyAOrg1 and CompanyA_network1, 
once this is run, now the CompanyBOrg1 is no longer part of the network CompanyA_network1 and would not receive any new updates.



## License
[GNU](https://choosealicense.com/licenses/agpl-3.0/#)
