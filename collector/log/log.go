package log

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	GetModel() model.Model
	GetInstanceId() string
	GetInspectionId() string
	GetScrapeBegin() (time.Time, error)
	GetScrapeEnd() (time.Time, error)
}

type LogCollector struct {
	Options
}

func New(opts Options) *LogCollector {
	return &LogCollector{opts}
}

func (c *LogCollector) Collect() error {
	begin, err := c.GetScrapeBegin()
	if err != nil {
		return err
	}
	end, err := c.GetScrapeEnd()
	if err != nil {
		return err
	}

	home := c.GetHome()
	instance := c.GetInstanceId()
	inspection := c.GetInspectionId()

	c.GetModel().UpdateInspectionMessage(inspection, "collecting log...")

	cmd := exec.Command(
		path.Join(home, "bin", "spliter"),
		fmt.Sprintf("--src=%s", path.Join(home, "remote-log", instance)),
		fmt.Sprintf("--dst=%s", path.Join(home, "inspection", inspection, "log")),
		fmt.Sprintf("--begin=%s", begin.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("split log:", err)
		return err
	}

	return nil
}
