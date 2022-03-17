# docker build . -t hub.pingcap.net/clinic/centos7_golang-1.17:latest
FROM hub.pingcap.net/jenkins/centos7_jenkins

USER root
WORKDIR /root

RUN yum remove git -y \
       && curl https://setup.ius.io | sh || true
RUN yum install -y git222 python-pip python3-pip

RUN pip install s3cmd requests && pip3 install requests &&  pip3 install qiniu

ENV GOLANG_VERSION 1.17.7
ENV GOLANG_DOWNLOAD_URL https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 02b111284bedbfa35a7e5b74a06082d18632eff824fd144312f6063943d49259

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
