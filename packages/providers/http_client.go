package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type jsonHTTPClient struct {
	client *http.Client
}

func newJSONHTTPClient(client *http.Client) jsonHTTPClient {
	if client == nil {
		client = http.DefaultClient
	}

	return jsonHTTPClient{client: client}
}

func (c jsonHTTPClient) doJSONRequest(req *http.Request, target any) error {
	_, err := c.doJSONRequestWithRaw(req, target)
	return err
}

func (c jsonHTTPClient) doJSONRequestWithRaw(req *http.Request, target any) ([]byte, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(raw))
	}

	if target == nil {
		return raw, nil
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return raw, nil
}

func newJSONRequest(method, rawURL string, body any) (*http.Request, error) {
	var reader io.Reader = http.NoBody
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
