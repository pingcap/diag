# Build DEB

> Note:
> 
> Need to be executed on a ubuntu or debian machine

## Installation dependency package

```
# Install make
[ubuntu@ip] sudo apt-get install make

# Install golang
[ubuntu@ip] cd ~
[ubuntu@ip] wget https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz
[ubuntu@ip] tar xf go1.12.7.linux-amd64.tar.gz
[ubuntu@ip] export PATH=$PATH:/root/go/bin # or add to /etc/bashrc、~/.bashrc

# Install  nodejs
[ubuntu@ip] sudo apt-get install curl
[ubuntu@ip] curl -sL https://deb.nodesource.com/setup_12.x | sudo -E bash -
[ubuntu@ip] sudo apt-get install nodejs

# Install yarn
[ubuntu@ip] curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
[ubuntu@ip] echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
[ubuntu@ip] sudo apt-get update && sudo apt-get install --no-install-recommends yarn

# Install git
[ubuntu@ip] sudo apt install git
```

## Start Build DEB package

```
[ubuntu@ip] cd ~
[ubuntu@ip] git clone https://github.com/pingcap/tidb-foresight.git
[ubuntu@ip] cd tidb-foresight/debbuild
[ubuntu@ip] sh generate.sh
```

## Get DEB package
After the build is complete, you can get the deb package in the `tidb-foresight/debbuild` derectory.
