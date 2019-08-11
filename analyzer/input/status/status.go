package status

// Status for each collect item, the key is the item name
type StatusMap map[string]Status

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Return the value (Status struct) for a specified item
func (s *StatusMap) Get(item string) Status {
	return (*s)[item]
}
