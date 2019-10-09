#!/usr/bin/env python

print("""To start tidb-foresight (will listen on port 9527):
    systemctl start foresight-9527
    systemctl start influxd-9528
    systemctl start prometheus-9529\n""")
print("""View the log as follows:
    journalctl -u foresight-9527
    journalctl -u influxd-9528
    journalctl -u prometheus-9529""")
