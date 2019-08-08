Summary: Tidb diagnostic tool
Name: tidb-foresight
Version: alpha
Release: 1
License: GPL
Group: Applications/Server
Distribution: Linux
Vendor: PingCAP
%description
tidb-foresight is a web-based tidb cluster diagnostic tool
%prep
rm -rf %{_builddir}/*
rm -rf %{_buildrootdir}/*
cd %{_sourcedir}/
rm -rf prometheus-2.8.1.linux-amd64.tar.gz influxdb-1.7.7-static_linux_amd64.tar.gz graphviz.tar.gz
if [ ! -d './tidb-foresight' ];then
	echo "There is no tidb-foresight in ~/rpmbuild/SOURCES/"
	exit 2
fi
wget https://github.com/prometheus/prometheus/releases/download/v2.8.1/prometheus-2.8.1.linux-amd64.tar.gz
wget https://dl.influxdata.com/influxdb/releases/influxdb-1.7.7-static_linux_amd64.tar.gz
wget https://graphviz.gitlab.io/pub/graphviz/stable/SOURCES/graphviz.tar.gz

tar xf influxdb-1.7.7-static_linux_amd64.tar.gz
tar xf prometheus-2.8.1.linux-amd64.tar.gz
tar xf graphviz.tar.gz
cp -r %{_sourcedir}/tidb-foresight %{_builddir}/
mv %{_sourcedir}/influxdb-1.7.7-1 %{_builddir}/
mv %{_sourcedir}/prometheus-2.8.1.linux-amd64 %{_builddir}/
mv %{_sourcedir}/graphviz-2.40.1 %{_builddir}/
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/
%build
cd %{_builddir}/tidb-foresight/api
go build
cd %{_builddir}/tidb-foresight/analyzer
go build
cd %{_builddir}/tidb-foresight/spliter
go build
cd %{_builddir}/tidb-foresight/syncer
go build
cd %{_builddir}/tidb-foresight/web
yarn && yarn build
cd %{_builddir}/graphviz-2.40.1
./configure --prefix=%{_builddir}/graphviz --enable-static=yes
make
make install

%install
# install foresight
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/web
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/systemd/system/
cp -r %{_builddir}/tidb-foresight/api/tidb-foresight %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/
cp -r %{_builddir}/tidb-foresight/analyzer/analyzer %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/
cp -r %{_builddir}/tidb-foresight/spliter/spliter %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/
cp -r %{_builddir}/tidb-foresight/syncer/syncer %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/
cp -r %{_builddir}/tidb-foresight/collector %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/
ln -s %{_builddir}/bin/collector/collect %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/collect
cp -r %{_builddir}/tidb-foresight/pioneer/pioneer.py %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/bin/pioneer
cp -r %{_builddir}/tidb-foresight/web/dist/* %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/tidb-foresight/web/
cat>%{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/systemd/system/foresight.service<<EOF
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

# install influxdb
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/log
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/conf
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/logrotate.d
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/var/lib/influxdb/meta
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/var/lib/influxdb/data
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/var/lib/influxdb/wal
cat>%{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/systemd/system/influxd.service<<EOF
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
Alias=influxd.service
EOF

cp -r %{_builddir}/influxdb-1.7.7-1/influx %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin/
cp -r %{_builddir}/influxdb-1.7.7-1/influxd %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin/
cp -r %{_builddir}/influxdb-1.7.7-1/influx_inspect %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin/
cp -r %{_builddir}/influxdb-1.7.7-1/influx_stress %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin/
cp -r %{_builddir}/influxdb-1.7.7-1/influx_tsm %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/bin/
cp -r %{_builddir}/influxdb-1.7.7-1/influxdb.conf %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/influxdb/conf/
cat>%{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/logrotate.d/influxdb<<EOF
/usr/local/influxdb/log/influxd.log {
    daily
    rotate 7
    missingok
    dateext
    copytruncate
    compress
}
EOF

# install prometheus
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/bin
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/conf
mkdir -p %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/data
cp -r %{_builddir}/prometheus-2.8.1.linux-amd64/prometheus %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/bin/
cp -r %{_builddir}/prometheus-2.8.1.linux-amd64/prometheus.yml %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/conf/
cat>>%{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/prometheus/conf/prometheus.yml<<EOF
remote_read:
  - url: "http://localhost:8086/api/v1/prom/read?db=inspection"
    read_recent: true

remote_write:
  - url: "http://localhost:8086/api/v1/prom/write?db=inspection"
EOF

cat>%{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/etc/systemd/system/prometheus.service<<EOF
# If you modify this, please also make sure to edit init.sh

[Unit]
Description=prometheus service
After=syslog.target network.target remote-fs.target nss-lookup.target

[Service]
LimitNOFILE=1000000
#LimitCORE=infinity
LimitSTACK=10485760
User=tidb
ExecStart=/usr/local/prometheus/bin/prometheus --web.listen-address=:8080 --storage.tsdb.path=/usr/local/prometheus/data --config.file=/usr/local/prometheus/conf/prometheus.yml
Restart=always
RestartSec=15s

[Install]
WantedBy=multi-user.target
EOF

# install graphviz
cp -r %{_builddir}/graphviz %{_buildrootdir}/%{name}-%{version}-%{release}.%{_build_arch}/usr/local/graphviz

%files
# foresight
/usr/local/
/etc/
/var/

%pre
grep -w tidb /etc/shadow > /dev/null
if [ $? != 0 ]; then
	useradd tidb
fi
grep -w influxdb /etc/shadow > /dev/null
if [ $? != 0 ]; then
	useradd influxdb
fi
%post
systemctl daemon-reload
chown -R tidb:tidb /usr/local/tidb-foresight
chown -R tidb:tidb /usr/local/prometheus
chown -R influxdb:influxdb /usr/local/influxdb
chown -R influxdb:influxdb /var/lib/influxdb
echo 'export PATH=$PATH:/usr/local/graphviz/bin' >> /etc/bashrc
echo 'export PATH=$PATH:/usr/local/graphviz/bin' >> ~/.bashrc
source ~/.bashrc
%preun
systemctl stop foresight.service
systemctl stop prometheus.service
systemctl stop influxd.service
%postun
# uninstall foresight
rm -rf /usr/local/tidb-foresight/
rm -rf /etc/systemd/system/foresight.service

# uninstall prometheus
rm -rf /usr/local/prometheus
rm -rf /etc/systemd/system/prometheus.service

# uninstall influxdb
rm -rf /usr/local/influxdb
rm -rf /etc/logrotate.d/influxdb
rm -rf /var/lib/influxdb
rm -rf /etc/systemd/system/influxd.service
rm -rf /usr/local/graphviz
sed -i '/graphviz/d' /etc/bashrc

%clean
rm -rf %{_buildrootdir}/*
rm -rf %{_builddir}/*
