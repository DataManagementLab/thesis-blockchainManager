# Blockchain Enabler Platform

This project is about a blockchain platform which would allow the user to create private blockchain with least hassle.

## Description 

Welcome to the Blockchain enabler Platform which makes handling and working with private blockchain easier and intuitive for the user. 
Current implementation would include [Hyperledger fabric](https://www.hyperledger.org/use/fabric) and [Ethereum](https://ethereum.org/en/)
The user can create and manage multiple blockchain networks using the platform.

Currently the user can create the blockchain network, invite other parties to this network and also leave the network.
All of this is done and the user can use the docker volumes or decide not using them.

## Table of Contents

## Requirements
The project requires some dependencies to be installed before using this platform.

1. [Docker](https://docs.docker.com/get-docker/)
2. [Docker compose](https://docs.docker.com/compose/install/)
3. [Golang](https://go.dev/doc/install)

## Setting up Project



## How to use the project

The platform is currently a command line which handles every thing from the network initialization, creation, joining to leaving.
The commands are made intuitive for the user to understand.

As mentioned the process is divided into steps so first step being the initialization of the network.
This includes different tasks while initializing the network.

This is done using the init command.
```cli
go run main.go init kinshuk
```
```bash
Flags:
  -s, --basicSetup    Choose this to form a network without the orderer default: disabled
  -b, --blockchain    Provide the Blockchain you would like to use options are [fabric geth corda] (default "fabric")
  -n, --networkID     Provide the name for the network. (default "kinshuk_network1")
  -o, --orgName       Provide the name for the organization default value org1. (default "Org1")
  -v, --useVolume     enable or disable the use of Volume default: disabled
```

create network
```cli
go run main.go create -u kinshuk -n kinshuk_network1
```
``` bash
  -b, --basicSetup      Function to enable or disable the use of Basic setup default: false
  -n, --netid string    Provide the network id of the network you want to run.
  -u, --userId string   Provide the user Id for the network you want to run.
  -v, --useVolume       enable or disable the use of Volume default: disabled
 ```

There are two types of network you can initialize
1. Creating the folder 

## License
[GNU](https://choosealicense.com/licenses/agpl-3.0/#)
