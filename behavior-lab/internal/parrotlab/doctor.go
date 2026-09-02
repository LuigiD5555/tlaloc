package parrotlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DoctorReport is a pre-flight check of the model endpoint.
type DoctorReport struct {
	Endpoint       string   `json:"endpoint"`
	Reachable      bool     `json:"reachable"`
	Models         []string `json:"models"`
	RequestedModel string   `json:"requested_model"`
	RequestedFound bool     `json:"requested_found"`
	Notes          []string `json:"notes"`
}

// Doctor queries {endpoint}/models and checks the configured model is served.
func Doctor(ctx context.Context, exp *Experiment) (DoctorReport, error) {
	report := DoctorReport{Endpoint: exp.Model.Endpoint, RequestedModel: exp.Model.ID}
	url := strings.TrimRight(exp.Model.Endpoint, "/") + "/models"

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return report, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		report.Notes = append(report.Notes, fmt.Sprintf("endpoint unreachable: %v", err))
		return report, nil
	}
	defer response.Body.Close()
	report.Reachable = response.StatusCode/100 == 2
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		report.Notes = append(report.Notes, "could not parse /models response")
		return report, nil
	}
	for _, model := range payload.Data {
		report.Models = append(report.Models, model.ID)
		if model.ID == exp.Model.ID {
			report.RequestedFound = true
		}
	}
	if exp.Model.ID == "" {
		report.Notes = append(report.Notes, "MODEL.json id is empty")
	} else if !report.RequestedFound {
		report.Notes = append(report.Notes, fmt.Sprintf("configured model %q not served by endpoint", exp.Model.ID))
	}
	if exp.Model.Temperature != 0 {
		report.Notes = append(report.Notes, "MODEL.json temperature is not 0 (R0 requires 0)")
	}
	return report, nil
}
