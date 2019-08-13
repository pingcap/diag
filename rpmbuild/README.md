# Build RPM

## Installation dependency package

```
# Install rpm-build
[root@ip] yum install -y rpm-build

# Install static compilation dependencies
[root@ip] yum install glibc-static libstdc++-static -y

# Install golang
[root@ip] cd ~
[root@ip] wget https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz
[root@ip] tar xf go1.12.7.linux-amd64.tar.gz
[root@ip] export PATH=$PATH:/root/go/bin # 或者加到 /etc/bashrc、~/.bashrc 等

# Install nvm
[root@ip] curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.34.0/install.sh | bash
[root@ip] source ~/.bashrc

# Install node
[root@ip] nvm install node

# Install yarn
[root@ip] curl -o- -L https://yarnpkg.com/install.sh | bash
[root@ip] source ~/.bashrc
```

## Build directory

```
[root@ip] cd ~
[root@ip] rpmbuild tidb-foresight
```

## Setting configuration
1. git clone tidb-foresight and put it in `~/rpmbuild/SOURCES/` directory.
2. Put the `tidb-foresight.spec` in the `tidb-foresight/rpmbuild` directory into the `~/rpmbuild/SPECS` directory.

## Begin build RPM

```
[root@ip] cd ~/rpmbuild
[root@ip] rpmbuild -bb SPECS/tidb-foresight.spec
```

## Get RPM
When the build is complete, you can get the rpm package in the `~/rpmbuild/RPMS` directory.

# Build DEB for ubuntu and debain

## Install alien

```
sudo apt install -y alien
```

## Convert RPM to DEB

```
sudo alien -k --scripts tidb-foresight-xxx.x86_64.rpm
```
