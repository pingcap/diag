package metric

import (
	"encoding/json"
	"strconv"
)

/*
 * The concept and definion of Matrix, Series, and Point comes frome prometheus
 */

// pointT represents a single data point for a given timestamp.
type pointT struct {
	T int64
	V float64
}

// UnmarshalJSON unmarshals data to a point
func (p *pointT) UnmarshalJSON(data []byte) error {
	var a [2]interface{}
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	val, err := strconv.ParseFloat(a[1].(string), 64)
	if err != nil {
		return err
	}

	p.T = int64(a[0].(float64))
	p.V = val

	return nil
}
