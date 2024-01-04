package spacetraders

import (
	"context"
	"encoding/json"
)

type ClientConfig struct {
	APIUrl string `envconfig:"API_URL" default:"https://api.spacetraders.io/v2"`
}

type ServiceStatus struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	ResetDate string `json:"resetDate"`
}

func NewInitializedClientFromConfig(config *ClientConfig) (client ClientWithResponsesInterface, status *ServiceStatus, err error) {
	client, err = NewClientWithResponses(config.APIUrl)
	if err != nil {
		return nil, nil, err
	}

	statusResp, err := client.GetStatusWithResponse(context.Background())
	if err != nil {
		return nil, nil, err
	}

	if statusResp.StatusCode() != 200 {
		return nil, nil, NewAPIError(statusResp.StatusCode(), statusResp.Body)
	}

	if err := json.Unmarshal(statusResp.Body, status); err != nil {
		return nil, nil, err
	}

	return client, status, nil
}
