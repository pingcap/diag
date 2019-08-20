package report

type Item struct {
	InspectionId string `json:"-"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Messages     string `json:"messages"`
}

func (m *report) ClearInspectionItem(inspectionId string) error {
	return m.db.Delete(&Item{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionItem(item *Item) error {
	return m.db.Create(item).Error()
}
