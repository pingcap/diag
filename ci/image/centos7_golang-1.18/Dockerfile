# docker build . -t hub.pingcap.net/clinic/centos7_golang-1.21:latest
FROM hub.pingcap.net/jenkins/centos7_jenkins

USER root
WORKDIR /root

RUN yum remove git -y \
       && curl https://setup.ius.io | sh || true
RUN yum install -y git222 python-pip python3-pip

RUN pip install s3cmd requests && pip3 install requests &&  pip3 install qiniu

ENV GOLANG_VERSION 1.21
ENV GOLANG_DOWNLOAD_URL https://go.dev/dl/go1.21.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 e85278e98f57cdb150fe8409e6e5df5343ecb13cebf03a5d5ff12bd55a80264f

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

RUN mkdir /go && chown jenkins:jenkins /go

RUN curl -fsSL "http://pingcap-dev.hk.ufileos.com/jenkins/jenkins-slave-docker-sqllogictest.tar.gz" | tar xz -C "/git" \
        && chown -R jenkins:jenkins /git

ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH

USER jenkins
WORKDIR /home/jenkins
RUN curl --proto '=https' --tlsv1.2 -sSf https://tiup-mirrors.pingcap.com/install.sh | sh
ENV PATH /home/jenkins/.tiup/bin:$PATH
