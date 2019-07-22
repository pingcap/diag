package report

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type Item struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Messages []string `json:"messages"`
}

func (r *Report) loadItems() error {
	items := []*Item{}

	rows, err := r.db.Query("SELECT name,status,message FROM inspection_items WHERE inspection = ?", r.inspectionId)
	if err != nil {
		log.Error("failed to call db.Query:", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		message := ""
		item := Item{Messages: []string{}}
		err = rows.Scan(&item.Name, &item.Status, &message)
		if err != nil {
			log.Error("failed to call db.Query:", err)
			return err
		}
		if message != "" {
			item.Messages = strings.Split(message, "|")
		}
		items = append(items, &item)
	}

	r.Items = items
	return nil
}

func (r *Report) itemReady(name string) bool {
	items := r.Items.([]*Item)
	for _, item := range items {
		if item.Name == name {
			return item.Status == "success" || item.Status == "error"
		}
	}
	return false
}
