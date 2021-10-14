package command

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

type Histroy struct {
	file string
	list []string
}

func loadHistroy() (*Histroy, error) {
	dir := os.Getenv("TIUP_COMPONENT_DATA_DIR")
	if dir == "" {
		dir = path.Join(os.Getenv("HOME"), ".clinic")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	file := path.Join(dir, "history.txt")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	list := strings.Split(string(data), "\n")
	return &Histroy{file, list}, nil
}

func (h *Histroy) Push(url string) {
	h.list = append([]string{url}, h.list...)
}

func (h *Histroy) Store() error {
	list := h.list
	if len(list) > 10 {
		list = list[:10]
	}
	return os.WriteFile(h.file, []byte(strings.Join(list, "\n")), 0664)
}

func (h *Histroy) PrintList() {
	for _, url := range h.list {
		fmt.Println(url)
	}
}

func newHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "show upload history",
		RunE: func(cmd *cobra.Command, args []string) error {
			his, err := loadHistroy()
			if err != nil {
				return err
			}
			his.PrintList()
			return nil
		},
	}

	return cmd
}
