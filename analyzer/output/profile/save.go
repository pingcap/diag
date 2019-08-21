package profile

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type saveProfileTask struct {
	inspectionId string
	bin          string
	m            *boot.Model
}

func SaveProfile() *saveProfileTask {
	return &saveProfileTask{}
}

// Save and make svg of each profile collected
func (t *saveProfileTask) Run(c *boot.Config, m *boot.Model) {
	// Setup config
	t.inspectionId = c.InspectionId
	t.bin = c.Bin
	t.m = m

	// eg. pd -> 172.16.5.7:2379 -> cpu.pb.gz
	comps, err := ioutil.ReadDir(path.Join(c.Src, "profile"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read dir:", err)
		}
		return
	}

	for _, comp := range comps {
		addrs, err := ioutil.ReadDir(path.Join(c.Src, "profile", comp.Name()))
		if err != nil {
			log.Error("read dir:", err)
			return
		}

		for _, addr := range addrs {
			if err = t.copy(
				path.Join(c.Src, "profile", comp.Name(), addr.Name()),
				path.Join(c.Profile, comp.Name()+"-"+addr.Name(), "meta"),
			); err != nil {
				log.Error("copy:", err)
				return
			}

			t.flame(
				path.Join(c.Profile, comp.Name()+"-"+addr.Name(), "meta"),
				path.Join(c.Profile, comp.Name()+"-"+addr.Name(), "flame"),
			)
		}
	}
}

func (t *saveProfileTask) copy(src, dst string) error {
	f, err := os.Stat(src)
	if err != nil {
		fmt.Println("stat:", err)
		return err
	}
	switch mode := f.Mode(); {
	case mode.IsDir():
		if err = os.MkdirAll(dst, os.ModePerm); err != nil {
			fmt.Println("mkdir:", err)
			return err
		}
		files, err := ioutil.ReadDir(src)
		if err != nil {
			fmt.Println("readdir:", err)
			return err
		}
		for _, f := range files {
			if err = t.copy(path.Join(src, f.Name()), path.Join(dst, f.Name())); err != nil {
				return err
			}
		}
	case mode.IsRegular():
		from, err := os.Open(src)
		if err != nil {
			fmt.Println("open:", err)
			return err
		}
		defer from.Close()
		to, err := os.Create(dst)
		if err != nil {
			fmt.Println("create:", err)
			return err
		}
		defer to.Close()

		if _, err := io.Copy(to, from); err != nil {
			log.Error("io.Copy:", err)
			return err
		}
	}
	return nil
}

func (t *saveProfileTask) flame(src, dst string) {
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		log.Error("mkdir:", err)
		return
	}

	profiles, err := ioutil.ReadDir(src)
	if err != nil {
		log.Error("read dir:", err)
		return
	}

	for _, profile := range profiles {
		if strings.HasSuffix(profile.Name(), ".pb.gz") {
			if err := t.flameGo(path.Join(src, profile.Name()), path.Join(dst, profile.Name()+".svg")); err != nil {
				log.Error("make flame:", err)
				t.m.InsertSymptom(
					"exception",
					fmt.Sprintf("making flame for %s", profile.Name()),
					"this error is not about the tidb cluster you are running, it's about tidb-foresight itself",
				)
			}
		} else if profile.Name() == "perf.data" {
			if err := t.flameRust(path.Join(src, profile.Name()), path.Join(dst, profile.Name()+".svg")); err != nil {
				log.Error("make flame:", err)
				t.m.InsertSymptom(
					"exception",
					fmt.Sprintf("making flame for %s", profile.Name()),
					"this error is not about the tidb cluster you are running, it's about tidb-foresight itself",
				)
			}
		}
	}
}

func (t *saveProfileTask) flameGo(src, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		fmt.Println("create:", err)
		return err
	}
	defer f.Close()

	cmd := exec.Command("go", "tool", "pprof", "--svg", src)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)

	if err = cmd.Run(); err != nil {
		log.Error("exec:", err)
		return err
	}
	return nil
}

func (t *saveProfileTask) flameRust(src, dst string) error {
	df, err := os.Create(dst)
	if err != nil {
		fmt.Println("create:", err)
		return err
	}
	defer df.Close()

	script := exec.Command("perf", "script", fmt.Sprintf("--input=%s", src))
	fold := exec.Command(path.Join(t.bin, "fold-tikv-threads-perf.pl"), "--threads", ".*")
	collapse := exec.Command(path.Join(t.bin, "stackcollapse-perf.pl"))
	flamegraph := exec.Command(path.Join(t.bin, "flamegraph.pl"))

	script.Stderr = os.Stderr
	if fold.Stdin, err = script.StdoutPipe(); err != nil {
		log.Error("pipe stdout:", err)
		return err
	}
	defer fold.Stdin.(io.ReadCloser).Close()

	fold.Stderr = os.Stderr
	if collapse.Stdin, err = fold.StdoutPipe(); err != nil {
		log.Error("pipe stdout:", err)
		return err
	}
	defer collapse.Stdin.(io.ReadCloser).Close()

	collapse.Stderr = os.Stderr
	if flamegraph.Stdin, err = collapse.StdoutPipe(); err != nil {
		log.Error("pipe stdout:", err)
		return nil
	}
	defer flamegraph.Stdin.(io.ReadCloser).Close()

	flamegraph.Stderr = os.Stderr
	flamegraph.Stdout = df

	if err = utils.StartCommands(script, fold, collapse, flamegraph); err != nil {
		log.Error("exec:", err)
		return err
	}
	if err = utils.WaitCommands(script, fold, collapse, flamegraph); err != nil {
		log.Error("exec:", err)
		return err
	}

	return nil
}
