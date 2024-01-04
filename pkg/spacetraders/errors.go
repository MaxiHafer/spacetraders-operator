package spacetraders

import "encoding/json"

var _ error = (*APIError)(nil)

func NewAPIError(statusCode int, body []byte) error {
	err := &APIError{
		StatusCode: statusCode,
	}

	if err := json.Unmarshal(body, &err.Message); err != nil {
		return err
	}

	return err
}

type APIError struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
}

func (a APIError) Error() string {
	return a.Message
}
