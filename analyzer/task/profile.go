package task

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/utils"
	log "github.com/sirupsen/logrus"
)

type SaveProfileTask struct {
	BaseTask
}

func SaveProfile(base BaseTask) Task {
	return &SaveProfileTask{base}
}

func (t *SaveProfileTask) Run() error {
	if !t.data.args.Collect(ITEM_PROFILE) || t.data.status[ITEM_PROFILE].Status != "success" {
		return nil
	}

	// eg. pd -> 172.16.5.7:2379 -> cpu.pb.gz
	comps, err := ioutil.ReadDir(path.Join(t.src, "profile"))
	if err != nil {
		log.Error("read dir:", err)
		return err
	}

	for _, comp := range comps {
		addrs, err := ioutil.ReadDir(path.Join(t.src, "profile", comp.Name()))
		if err != nil {
			log.Error("read dir:", err)
			return err
		}

		for _, addr := range addrs {
			if err = t.copy(
				path.Join(t.src, "profile", comp.Name(), addr.Name()),
				path.Join(t.profile, comp.Name()+"-"+addr.Name(), "meta"),
			); err != nil {
				log.Error("copy:", err)
				return err
			}

			if err = t.flame(
				path.Join(t.profile, comp.Name()+"-"+addr.Name(), "meta"),
				path.Join(t.profile, comp.Name()+"-"+addr.Name(), "flame"),
			); err != nil {
				log.Error("make flame:", err)
				return err
			}
		}
	}

	return nil
}

func (t *SaveProfileTask) copy(src, dst string) error {
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

		_, err = io.Copy(to, from)
		if err != nil {
			log.Error("io.Copy:", err)
			return err
		}
	}
	return nil
}

func (t *SaveProfileTask) flame(src, dst string) error {
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		log.Error("mkdir:", err)
		return err
	}

	profiles, err := ioutil.ReadDir(src)
	if err != nil {
		log.Error("read dir:", err)
		return err
	}

	for _, profile := range profiles {
		if strings.HasSuffix(profile.Name(), ".pb.gz") {
			err = t.flameGo(path.Join(src, profile.Name()), path.Join(dst, profile.Name()+".svg"))
			if err != nil {
				log.Error("make flame:", err)
				t.InsertSymptom(
					"exception",
					fmt.Sprintf("error on making flame for %s", profile.Name()),
					"this error is not about the tidb cluster you are running, it's about tidb-foresight itself",
				)
				return nil
			}
		} else if profile.Name() == "perf.data" {
			err = t.flameRust(path.Join(src, profile.Name()), path.Join(dst, profile.Name()+".svg"))
			if err != nil {
				log.Error("make flame:", err)
				t.InsertSymptom(
					"exception",
					fmt.Sprintf("error on making flame for %s", profile.Name()),
					"this error is not about the tidb cluster you are running, it's about tidb-foresight itself",
				)
				return nil
			}
		}
	}
	return nil
}

func (t *SaveProfileTask) flameGo(src, dst string) error {
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

func (t *SaveProfileTask) flameRust(src, dst string) error {
	df, err := os.Create(dst)
	if err != nil {
		fmt.Println("create:", err)
		return err
	}
	defer df.Close()

	script := exec.Command("perf", "script", fmt.Sprintf("--input=%s", src))
	collapse := exec.Command(path.Join(t.bin, "stackcollapse-perf.pl"))
	flamegraph := exec.Command(path.Join(t.bin, "flamegraph.pl"))

	script.Stderr = os.Stderr
	if collapse.Stdin, err = script.StdoutPipe(); err != nil {
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

	if err = utils.StartCommands(script, collapse, flamegraph); err != nil {
		log.Error("exec:", err)
		return err
	}
	if err = utils.WaitCommands(script, collapse, flamegraph); err != nil {
		log.Error("exec:", err)
		return err
	}

	return nil
}
