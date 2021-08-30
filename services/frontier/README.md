# Frontier
Frontier is the client facing API server for the [DigitalBits ecosystem](https://developer.digitalbits.io/guides/get-started/). It acts as the interface between [DigitalBits Core](https://github.com/xdbfoundation/DigitalBits) and applications that want to access the DigitalBits network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.

## Try it out
See Frontier in action by running your own DigitalBits node as part of the DigitalBits [testnet](https://github.com/xdbfoundation/docs/blob/master/guides/concepts/test-net.md). With our Docker quick-start image, you can be running your own fully functional node in around 20 minutes. See the [Quickstart Guide](internal/docs/quickstart.md) to get up and running.

## Prebuild software
DigitalBits.io publishes frontier packages to the cloudsmith.io repository https://cloudsmith.io/~xdb-foundation/repos/digitalbits-frontier/packages/

Packages available as:
   - .deb and .rpm packages and 
   - prebuild binaries for windows, linux and macos
   - docker image.  

## DEB-based

1. Configure digitalbits-frontier repository from cloudsmith.io:

        curl -1sLf 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/setup.deb.sh' | sudo -E bash

2. Install digitalbits-frontier package:

        sudo apt-get install digitalbits-frontier


## RPM-based
1. Configure digitalbits-frontier repository from cloudsmith.io:

        curl -1sLf 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/setup.rpm.sh' | sudo -E bash

2. Install digitalbits-frontier package:

        sudo yum install digitalbits-frontier


## Raw binaries

- MacOS

        curl -O 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/raw/files/frontier_${VERSION}_darwin-amd64.tar.gz'

- Linux

        curl -O 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/raw/files/frontier_${VERSION}_linux-amd64.tar.gz'

- Windows

        curl -O 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/raw/files/frontier_${VERSION}_windows-amd64.tar.gz'

- Linux (32-bit)

        curl -O 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/raw/files/frontier_${VERSION}_linux-386.tar.gz'
        

- Windows (32-bit)

        curl -O 'https://dl.cloudsmith.io/public/xdb-foundation/digitalbits-frontier/raw/files/frontier_${VERSION}_windows-386.tar.gz'

        

## Docker image

    docker pull docker.cloudsmith.io/xdb-foundation/digitalbits-frontier/digitalbits-frontier:latest


## Run a production server
If you're an administrator planning to run a production instance of Frontier as part of the public DigitalBits network, check out the detailed [Administration Guide](internal/docs/admin.md). It covers installation, monitoring, error scenarios and more.

## Contributing
As an open source project, development of Frontier is public, and you can help! We welcome new issue reports, documentation and bug fixes, and contributions that further the project roadmap. The [Development Guide](internal/docs/developing.md) will show you how to build Frontier, see what's going on behind the scenes, and set up an effective develop-test-push cycle so that you can get your work incorporated quickly.
