package report

type Item struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Messages []string `json:"messages"`
}
