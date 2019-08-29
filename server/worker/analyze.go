package worker

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	log "github.com/sirupsen/logrus"
)

func (w *worker) Analyze(inspectionId string) error {
	analyzer := path.Join(w.c.Home, "bin", "analyzer")
	cmd := exec.Command(
		analyzer,
		fmt.Sprintf("--home=%s", w.c.Home),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
	)
	cmd.Env = append(
		os.Environ(),
		"INFLUX_ADDR="+w.c.Influx.Endpoint,
		"PROM_ADDR="+w.c.Prometheus.Endpoint,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", analyzer, ": ", err)
		return err
	}
	return nil
}
