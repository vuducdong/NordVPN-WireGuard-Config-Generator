package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nordgen/internal/constants"
	"nordgen/internal/models"
)

type NordClient struct {
	httpClient *http.Client
}

func NewNordClient() *NordClient {
	var transport *http.Transport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	} else {
		transport = &http.Transport{}
	}
	transport.MaxIdleConns = 10
	transport.IdleConnTimeout = 30 * time.Second
	transport.MaxIdleConnsPerHost = 10

	return &NordClient{
		httpClient: &http.Client{
			Timeout:   25 * time.Second,
			Transport: transport,
		},
	}
}

func (c *NordClient) GetKey(token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, constants.CredsURL, nil)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte("token:" + token))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var payload struct {
		NordlynxPrivateKey string `json:"nordlynx_private_key"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	return payload.NordlynxPrivateKey, nil
}

func (c *NordClient) GetGeo() (float64, float64, error) {
	resp, err := c.httpClient.Get(constants.GeoURL)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var payload struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, err
	}

	return payload.Latitude, payload.Longitude, nil
}

func (c *NordClient) GetServers() ([]models.RawServer, error) {
	resp, err := c.httpClient.Get(constants.ServersURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var servers []models.RawServer
	if err := json.Unmarshal(body, &servers); err != nil {
		return nil, err
	}

	return servers, nil
}
