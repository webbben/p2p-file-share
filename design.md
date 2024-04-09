## Overview - A Distributed File Sharing System

This is a personal project I made to experiment with distributed systems and network programming. It is a distributed file sharing system, which allows you to install nodes on several of your computers on a local network, and for them to communicate with one-another, maintaining a "shared directory" that contains files propagated among the nodes.
If a file is added to the specified directory on one computer/node, then that file will be propagated to the other computers hosting nodes as well. And if a file is removed from the specified directory on one computer/node, it is removed from the other computers/nodes.

## Why make this?

My main motivation is to get better acquainted with network programming and experiment with different system architectures; Typically I've always made projects that follow the server -> client architecture, which usually does the job quite well. And perhaps for this type of file sharing system, it would work quite well too. However, the shortcoming of that architecture is that I need to have a centralized server that is always running, or else the file sharing system fails.

However, in practice, this isn't much of a concern - the scale of running a local network file sharing system is pretty small, so keeping a single computer running a central server is probably trivial. But, I thought - _wouldn't it be cool to make an overly complex distributed file sharing system?_ Of course it would! The theoretical benefit of making it distributed is the system no longer relies on a single central node to be a global server, but rather all the nodes on the system simultaneously can act as servers and clients, sending and receiving files as needed from whichever node on the system has the file it needs.

Beyond these things - I also just thought it sounds like a useful tool to have. I have a few different old laptops and computers lying around, so I thought it would be cool to be able to easily share files between them, rather than needing to use things like external hard drives/USB, 3rd party cloud providers, or other various ways to send files around.

## Technical Details

### Architecture

As mentioned above, this is a "distributed system" - meaning that rather than the classic model of a central server communicating with client(s), it is a network of "nodes", each of which act both as a server or a client, depending on the situation.

A distributed system is probably overkill for this project, since we could easily accomplish the same thing by having a server that serves files to the nodes that request them, and maintains the "shared files" on its own. But in theory, this system is a little more robust since it doesn't rely on a single node/server to keep the system running; as nodes come on or offline, they can communicate with other nodes on the network and receive the files they need.

### Communication

This system is "peer to peer", meaning that each node communicates directly to the other nodes, rather than going via a middleman server.
I decided to use TCP for communication between nodes, including "handshakes" that are used for peer discovery, as well as sending the actual files between nodes. I couldn't used HTTP for all the communication, as that's what I'm usually using with web servers and clients I build for web development, but I decided to step a level lower just to test the waters.

In practice, all the communication works pretty much the same; the TCP connections are passing "headers" that identify the purpose of the message, and then the file contents are copied over the TCP connection to the other nodes.

### Security

The main security implemented is the fact that nodes in the system will only be willing to communicate with other nodes that are on the same local subnet; if an IP address doesn't have the same subnet, then it won't even attempt to communicate with it. Additionally, before establishing connections with peers and exchanging files, both nodes need to perform a handshake where specific information is passed between the two nodes. Nodes that aren't trusted won't be included in the network.

### Consensus Algorithm

TODO - define the algorithm that will maintain consensus between the nodes and their files.

## Functional Specs

Below, I'll outline how I intend for this file share system to work from a user's perspective.

### Setting up a node

First, a user must set up nodes on at least two different computers that are all on the same local area network. This would just involve connecting two or more computers to the same internet router.
To actually set up the node, the user will:

-   download the code to their computer and run it
-   go through first time setup, which includes **defining the local directory for shared files**. (this directory should be empty).
    -   this step is required, or else the node will not be able to share files
-   leave the program running, so that the node is ready to communicate with other nodes on the network

### Adding a file to the network

Once there is at least one node set up, a user can start adding files to the network. (of course - with only one node on the network, this won't do much of anything).
To add a file to the network, simply move the file into the "shared file directory" that was specified during the node setup. The node will automatically detect this file and go about propagating it to the other nodes on the network.

### Removing a file from the network

Similar to adding a file, a user can remove a file simply by deleting it from the shared file directory of one of the nodes. The program will notice this change, and send a message to other nodes to delete the file as well.

### User interface

Although this program doesn't require user interaction for the file propagation between nodes, there is a user interface in the terminal. This UI will:

-   allow the user to change the configuration, such as the directory for the file sharing, or other meta data associated with the node.
-   show updates when files are sent or received from other nodes
-   let the user see when each file was last modified, and by which node.
-   let the user see a list of all known nodes, and which are currently online or offline, and other status information on the nodes in the network.
