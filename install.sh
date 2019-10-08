# !/bin/sh

yum install -y graphviz perf rsync golang

systemctl stop foresight-9527
systemctl stop influxd-9528
systemctl stop prometheus-9529

# mkdir -p /opt/tidb/tidb-foresight
# cp -r * /opt/tidb/tidb-foresight/
# mv /opt/tidb/tidb-foresight/*.service /etc/systemd/system/
# chown -R tidb:tidb /opt/tidb/tidb-foresight
# chmod 755 /opt/tidb/tidb-foresight/*

# systemctl daemon-reload
# systemctl start foresight-9527
# systemctl start influxd-9528
# systemctl start prometheus-9529

# echo """To start tidb-foresight (will listen on port 9527):
#         systemctl start foresight-9527
#         systemctl start influxd-9528
#         systemctl start prometheus-9529"""
# echo """View the log as follows:
#         journalctl -u foresight-9527
#         journalctl -u influxd-9528
#         journalctl -u prometheus-9529"""
