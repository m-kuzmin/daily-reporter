package response

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

type APIRequester struct {
	Client   http.Client
	Scheme   string
	Host     string
	BasePath string
}

func (r APIRequester) DoJSONEncoded(ctx context.Context, endpoint string, body json.RawMessage,
) (json.RawMessage, error) {
	url := url.URL{
		Scheme: r.Scheme,
		Host:   r.Host,
		Path:   path.Join(r.BasePath, endpoint),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), bytes.NewReader([]byte(body)))
	if err != nil {
		// Delegates the correctness of the request to the one who is making it. If they can't ensure the request will
		// be created, they should do it themselves.
		return json.RawMessage{}, fmt.Errorf("while constructing get request to /%s: %w", endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return json.RawMessage{}, fmt.Errorf("network error: %w", err)
	}

	body, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return json.RawMessage{}, fmt.Errorf("could not read response body %w", err)
	}

	var data struct {
		Ok bool `json:"ok"`
		APIError
		Result json.RawMessage `json:"result,omitempty"`
	}

	if err = json.Unmarshal(body, &data); err != nil {
		return data.Result, fmt.Errorf("parsing json response error: %w", err)
	}

	if !data.Ok {
		return json.RawMessage{}, APIError{
			ErrorCode:   data.ErrorCode,
			Description: data.Description,
			Parameters:  data.Parameters,
		}
	}

	return data.Result, nil
}

func (r APIRequester) DoURLEncoded(ctx context.Context, endpoint string, params url.Values) (json.RawMessage, error) {
	url := url.URL{
		Scheme:   r.Scheme,
		Host:     r.Host,
		Path:     path.Join(r.BasePath, endpoint),
		RawQuery: params.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		// Delegates the correctness of the request to the one who is making it. If they can't ensure the request will
		// be created, they should do it themselves.
		return json.RawMessage{}, fmt.Errorf("while constructing URL encoded get request to /%s: %w", endpoint, err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return json.RawMessage{}, fmt.Errorf("network error: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()

		return json.RawMessage{}, fmt.Errorf("could not read response body %w", err)
	}

	resp.Body.Close()

	var data struct {
		Ok bool `json:"ok"`
		APIError
		Result json.RawMessage `json:"result,omitempty"`
	}

	if err = json.Unmarshal(body, &data); err != nil {
		return data.Result, fmt.Errorf("parsing json response error: %w", err)
	}

	if !data.Ok {
		return json.RawMessage{}, APIError{
			ErrorCode:   data.ErrorCode,
			Description: data.Description,
			Parameters:  data.Parameters,
		}
	}

	return data.Result, nil
}
