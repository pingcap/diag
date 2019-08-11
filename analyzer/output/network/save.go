package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type saveNetworkTask struct{}

func SaveNetwork() *saveNetworkTask {
	return &saveNetworkTask{}
}

type netstat struct {
	connections int64
	recv        int64
	send        int64
	bad_seg     int64
	retrans     int64
}

// Parse and save network information from output of netstat -s
func (t *saveNetworkTask) Run(db *boot.DB, c *boot.Config) {
	netDir := path.Join(c.Src, "net")
	ls, err := ioutil.ReadDir(netDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read dir:", err)
		}
		return
	}
	for _, l := range ls {
		if !l.IsDir() {
			continue
		}
		host := l.Name()
		filePath := path.Join(netDir, host, "netstat")
		ns, err := newNetstat(filePath)
		if err != nil {
			return
		}
		if _, err := db.Exec(
			`INSERT INTO inspection_network(inspection, node_ip, connections, recv, send, bad_seg, retrans)
		  VALUES(?, ?, ?, ?, ?, ?, ?)`, c.InspectionId, host, ns.connections, ns.recv, ns.send, ns.bad_seg, ns.retrans,
		); err != nil {
			log.Error("db.Exec:", err)
			return
		}
	}

	return
}

var (
	connectionRE = regexp.MustCompile(`([0-9]+) connections established`)
	recvRE       = regexp.MustCompile(`([0-9]+) segments received`)
	sendRE       = regexp.MustCompile(`([0-9]+) segments send out`)
	badSegRE     = regexp.MustCompile(`([0-9]+) bad segments received`)
	retransRE    = regexp.MustCompile(`([0-9]+) segments retransmited`)
)

func newNetstat(filePath string) (*netstat, error) {
	ns := &netstat{}
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	ns.connections, err = parserItem(connectionRE, b)
	if err != nil {
		return nil, err
	}
	ns.recv, err = parserItem(recvRE, b)
	if err != nil {
		return nil, err
	}
	ns.send, err = parserItem(sendRE, b)
	if err != nil {
		return nil, err
	}
	ns.bad_seg, err = parserItem(badSegRE, b)
	if err != nil {
		return nil, err
	}
	ns.retrans, err = parserItem(retransRE, b)
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func parserItem(re *regexp.Regexp, b []byte) (int64, error) {
	if !re.Match(b) {
		return -1, fmt.Errorf("failed to find %s string", re.String())
	}
	match := re.FindSubmatch(b)
	str := string(match[1])
	item, err := strconv.Atoi(str)
	if err != nil {
		return -1, err
	}
	return int64(item), nil
}
