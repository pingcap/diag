#!/bin/bash

if [ $# -ne 1 ]; then
	echo '''Usage:
    Please enter deb package name, for example:
        sh generate.sh tidb-foresight-0alpha-1.amd64.deb
    Note: alpha must have a number in front of it, for example:
        0alpha'''
	exit 1
fi

workdir=`dirname $0`
cd $workdir
mkdir -p temp/tidb-foresight
currentdir=`pwd`
debdir="$currentdir/tidb-foresight"
tempdir="$currentdir/temp/"

mkdir -p $debdir/usr/local/tidb-foresight/bin
mkdir -p $debdir/usr/local/tidb-foresight/web
mkdir -p $debdir/usr/local/tidb-foresight/script
mkdir -p $debdir/etc/systemd/system
mkdir -p $debdir/usr/local/influxdb/bin
mkdir -p $debdir/usr/local/influxdb/log
mkdir -p $debdir/usr/local/influxdb/conf
mkdir -p $debdir/var/lib/influxdb/meta
mkdir -p $debdir/var/lib/influxdb/data
mkdir -p $debdir/var/lib/influxdb/wal
mkdir -p $debdir/usr/local/prometheus/bin
mkdir -p $debdir/usr/local/prometheus/conf
mkdir -p $debdir/usr/local/prometheus/data
mkdir -p $debdir/etc/logrotate.d

rm -rf prometheus-2.8.1.linux-amd64.tar.gz influxdb-1.7.7-static_linux_amd64.tar.gz stackcollapse-perf.pl flamegraph.pl fold-tikv-threads-perf.pl

cd ..
cp -r `ls |grep -v debbuild | xargs` debbuild/temp/tidb-foresight/
cd $tempdir
wget https://github.com/prometheus/prometheus/releases/download/v2.8.1/prometheus-2.8.1.linux-amd64.tar.gz
wget https://dl.influxdata.com/influxdb/releases/influxdb-1.7.7-static_linux_amd64.tar.gz
wget https://raw.githubusercontent.com/brendangregg/FlameGraph/master/stackcollapse-perf.pl
wget https://raw.githubusercontent.com/brendangregg/FlameGraph/master/flamegraph.pl
wget https://raw.githubusercontent.com/pingcap/tidb-inspect-tools/master/tracing_tools/perf/fold-tikv-threads-perf.pl

tar xf influxdb-1.7.7-static_linux_amd64.tar.gz
tar xf prometheus-2.8.1.linux-amd64.tar.gz
chmod +x *.pl

# Install foresight
cd $tempdir/tidb-foresight
make
cd $tempdir/tidb-foresight/web
yarn && yarn build

# if there is config file, cp it
if [ -e tidb-foresight.toml ];then
        cp tidb-foresight.toml $debdir/usr/local/tidb-foresight/
fi

cp -r $tempdir/tidb-foresight/bin/* $debdir/usr/local/tidb-foresight/bin/
cp -r $tempdir/stackcollapse-perf.pl $debdir/usr/local/tidb-foresight/bin/
cp -r $tempdir/flamegraph.pl $debdir/usr/local/tidb-foresight/bin/
cp -r $tempdir/fold-tikv-threads-perf.pl $debdir/usr/local/tidb-foresight/bin/
cp -r $tempdir/tidb-foresight/collector $debdir/usr/local/tidb-foresight/script/
cp -r $tempdir/tidb-foresight/pioneer/pioneer.py $debdir/usr/local/tidb-foresight/bin/pioneer
cp -r $tempdir/tidb-foresight/web/dist/* $debdir/usr/local/tidb-foresight/web/
cat>$debdir/etc/systemd/system/foresight-9527.service<<EOF
# If you modify this, please also make sure to edit init.sh

[Unit]
Description=web service
After=syslog.target network.target remote-fs.target nss-lookup.target

[Service]
LimitNOFILE=1000000
#LimitCORE=infinity
LimitSTACK=10485760
User=tidb
Group=tidb
ExecStart=/usr/local/tidb-foresight/bin/tidb-foresight --home=/usr/local/tidb-foresight
Restart=always
RestartSec=15s

[Install]
WantedBy=multi-user.target
EOF

# Install influxdb
cat>$debdir/etc/systemd/system/influxd-9528.service<<EOF
# If you modify this, please also make sure to edit init.sh

[Unit]
Description=InfluxDB is an open-source, distributed, time series database
Documentation=https://docs.influxdata.com/influxdb/
After=network-online.target

[Service]
LimitNOFILE=1000000
#LimitCORE=infinity
LimitSTACK=10485760
User=influxdb
Group=influxdb
ExecStart=/usr/local/influxdb/bin/influxd -config /usr/local/influxdb/conf/influxdb.conf
Restart=always
RestartSec=15s

[Install]
WantedBy=multi-user.target
Alias=influxd-9528.service
EOF

cat>>$debdir/etc/logrotate.d/influxdb<<EOF
# influxdb log
/usr/local/influxdb/log/influxd.log {
    daily
    rotate 7
    missingok
    dateext
    copytruncate
    compress
}
EOF

cp -r $tempdir/influxdb-1.7.7-1/influx $debdir/usr/local/influxdb/bin/
cp -r $tempdir/influxdb-1.7.7-1/influxd $debdir/usr/local/influxdb/bin/
cp -r $tempdir/influxdb-1.7.7-1/influx_inspect $debdir/usr/local/influxdb/bin/
cp -r $tempdir/influxdb-1.7.7-1/influx_stress $debdir/usr/local/influxdb/bin/
cp -r $tempdir/influxdb-1.7.7-1/influx_tsm $debdir/usr/local/influxdb/bin/
cp -r $tempdir/influxdb-1.7.7-1/influxdb.conf $debdir/usr/local/influxdb/conf/
sed -i 's/\# bind-address \= \"\:/bind-address = "127.0.0.1\:/g' $debdir/usr/local/influxdb/conf/influxdb.conf
sed -i 's/127.0.0.1\:8086/127.0.0.1\:9528/g' $debdir/usr/local/influxdb/conf/influxdb.conf

# Install prometheus
cp -r $tempdir/prometheus-2.8.1.linux-amd64/prometheus $debdir/usr/local/prometheus/bin/
cp -r $tempdir/prometheus-2.8.1.linux-amd64/prometheus.yml $debdir/usr/local/prometheus/conf/
cat>>$debdir/usr/local/prometheus/conf/prometheus.yml<<EOF
remote_read:
  - url: "http://localhost:9528/api/v1/prom/read?db=inspection"
    read_recent: true

remote_write:
  - url: "http://localhost:9528/api/v1/prom/write?db=inspection"
EOF

cat>$debdir/etc/systemd/system/prometheus-9529.service<<EOF
# If you modify this, please also make sure to edit init.sh

[Unit]
Description=prometheus service
After=syslog.target network.target remote-fs.target nss-lookup.target

[Service]
LimitNOFILE=1000000
#LimitCORE=infinity
LimitSTACK=10485760
User=tidb
ExecStart=/usr/local/prometheus/bin/prometheus --web.listen-address=127.0.0.1:9529 --storage.tsdb.path=/usr/local/prometheus/data --config.file=/usr/local/prometheus/conf/prometheus.yml
Restart=always
RestartSec=15s

[Install]
WantedBy=multi-user.target
EOF

cd $currentdir
sudo dpkg -b ./tidb-foresight $1
