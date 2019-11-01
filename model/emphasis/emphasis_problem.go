package emphasis

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/utils"
)

// The Problem in Emphasis.
type Problem struct {
	CreateTime   utils.NullTime   `json:"create_time"`
	InspectionId string           `json:"inspection_id"`
	Uuid         string           `json:"uuid" gorm:"PRIMARY_KEY"`
	RelatedGraph string           `json:"related_graph"` // Related Grafana Graph.
	Problem      utils.NullString `json:"problem"`       // Related problem, return json null to represent no problem here.
	Advise       string           `json:"advise"`
}

// This type will not be cast to gorm type.
type ProblemSymptomInner struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Value   string `json:"value"`
}

// This type will not be cast to gorm type.
type ProblemSymptom struct {
	RelatedGraph string               `json:"related_graph"`
	Prob         *ProblemSymptomInner `json:"problem"`
}

// The `Problem` will be cast to
// ```
// type ProblemSymptom struct {
//	 RelatedGraph string               `json:"related_graph"`
//	 Prob         *ProblemSymptomInner `json:"problem"`
// }
// ```
//
func (prob *Problem) ProblemToSymptom() *ProblemSymptom {

	symptom := &ProblemSymptom{}
	symptom.RelatedGraph = prob.RelatedGraph
	if !prob.Problem.Valid {
		symptom.Prob = nil
	} else {
		symptom.Prob = &ProblemSymptomInner{
			Message: prob.Advise,
			// TODO: Now forced warning
			Status: "alert",
			Value:  prob.Problem.String,
		}
	}
	return symptom
}

func ArrayToSymptoms(problem []*Problem) map[string]interface{} {
	// allocate memory for symptoms
	symptomArray := make([]*ProblemSymptom, len(problem), len(problem))
	warnings := make([]*ProblemSymptomInner, 0)
	for i, v := range problem {
		symptomArray[i] = v.ProblemToSymptom()
		// copy
		newInnerProb := *symptomArray[i].Prob
		newInnerProb.Message = fmt.Sprintf("[%s]: %s", v.RelatedGraph, newInnerProb.Message)
		if symptomArray[i].Prob != nil {
			warnings = append(warnings, symptomArray[i].Prob)
		}
	}
	return map[string]interface{}{
		"conclusion": warnings,
		"data":       symptomArray,
	}
}
