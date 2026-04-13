package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LREClient communicates with the LRE Performance Center REST API.
type LREClient struct {
	BaseURL    string // https://server:port/LoadTest/rest
	Domain     string
	Project    string
	httpClient *http.Client
	cookie     string // LWSSO_COOKIE_KEY
}

// NewLREClient creates a new LRE PC API client.
func NewLREClient(server, domain, project string) *LREClient {
	return &LREClient{
		BaseURL: server,
		Domain:  domain,
		Project: project,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate logs in and stores the session cookie.
func (c *LREClient) Authenticate(username, password string) error {
	url := fmt.Sprintf("%s/authentication-point/authenticate", c.BaseURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, http.NoBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "LWSSO_COOKIE_KEY" {
			c.cookie = cookie.Value
			return nil
		}
	}

	return fmt.Errorf("LWSSO_COOKIE_KEY not found in response")
}

// Logout ends the session.
func (c *LREClient) Logout() error {
	url := fmt.Sprintf("%s/authentication-point/logout", c.BaseURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, http.NoBody)
	if err != nil {
		return err
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.cookie = ""
	return nil
}

func (c *LREClient) addAuth(req *http.Request) {
	if c.cookie != "" {
		req.Header.Set("Cookie", fmt.Sprintf("LWSSO_COOKIE_KEY=%s", c.cookie))
	}
	req.Header.Set("Content-Type", "application/json")
}

func (c *LREClient) projectURL() string {
	return fmt.Sprintf("%s/domains/%s/projects/%s", c.BaseURL, c.Domain, c.Project)
}

func (c *LREClient) doJSON(method, url string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		return err
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LRE API error: %s %s returned %d: %s", method, url, resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// LRETest represents a test in LRE PC.
type LRETest struct {
	Name string `json:"Name"`
	ID   int    `json:"ID"`
}

// ListTests returns all tests in the project.
func (c *LREClient) ListTests() ([]LRETest, error) {
	var tests []LRETest
	err := c.doJSON(http.MethodGet, c.projectURL()+"/tests", nil, &tests)
	return tests, err
}

// LREGroup represents a group (scenario) in an LRE PC test.
type LREGroup struct {
	Name       string `json:"Name"`
	ID         int    `json:"ID"`
	VuserCount int    `json:"VuserCount"`
	ScriptID   int    `json:"ScriptId"`
}

// ListGroups returns all groups for a test.
func (c *LREClient) ListGroups(testID int) ([]LREGroup, error) {
	var groups []LREGroup
	url := fmt.Sprintf("%s/tests/%d/groups", c.projectURL(), testID)
	err := c.doJSON(http.MethodGet, url, nil, &groups)
	return groups, err
}

// CreateGroup creates a new group in a test.
func (c *LREClient) CreateGroup(testID int, group LREGroup) (*LREGroup, error) {
	url := fmt.Sprintf("%s/tests/%d/groups", c.projectURL(), testID)
	var result LREGroup
	err := c.doJSON(http.MethodPost, url, group, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateGroup updates an existing group.
func (c *LREClient) UpdateGroup(testID, groupID int, group LREGroup) error {
	url := fmt.Sprintf("%s/tests/%d/groups/%d", c.projectURL(), testID, groupID)
	return c.doJSON(http.MethodPut, url, group, nil)
}

// LRERuntimeSettings holds runtime settings for a group.
type LRERuntimeSettings struct {
	Pacing LREPacing `json:"Pacing"`
}

// LREPacing holds pacing configuration.
type LREPacing struct {
	Type     string `json:"Type"`     // "ConstantPacing"
	MinDelay int    `json:"MinDelay"` // pacing in ms
	MaxDelay int    `json:"MaxDelay"` // same as MinDelay for constant
}

// GetRuntimeSettings gets runtime settings for a group.
func (c *LREClient) GetRuntimeSettings(testID, groupID int) (*LRERuntimeSettings, error) {
	url := fmt.Sprintf("%s/tests/%d/groups/%d/runtime-settings", c.projectURL(), testID, groupID)
	var settings LRERuntimeSettings
	err := c.doJSON(http.MethodGet, url, nil, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateRuntimeSettings updates runtime settings for a group.
func (c *LREClient) UpdateRuntimeSettings(testID, groupID int, settings LRERuntimeSettings) error {
	url := fmt.Sprintf("%s/tests/%d/groups/%d/runtime-settings", c.projectURL(), testID, groupID)
	return c.doJSON(http.MethodPut, url, settings, nil)
}

// LREScheduler holds scheduler settings for a test.
type LREScheduler struct {
	RampUpType       string `json:"RampUpType"`
	Duration         int    `json:"Duration"`
	RampUpAmount     int    `json:"RampUpAmount"`
	RampUpInterval   int    `json:"RampUpInterval"`
	RampDownAmount   int    `json:"RampDownAmount"`
	RampDownInterval int    `json:"RampDownInterval"`
}

// GetScheduler gets the scheduler settings for a test.
func (c *LREClient) GetScheduler(testID int) (*LREScheduler, error) {
	url := fmt.Sprintf("%s/tests/%d/scheduler", c.projectURL(), testID)
	var scheduler LREScheduler
	err := c.doJSON(http.MethodGet, url, nil, &scheduler)
	if err != nil {
		return nil, err
	}
	return &scheduler, nil
}

// UpdateScheduler updates the scheduler settings for a test.
func (c *LREClient) UpdateScheduler(testID int, scheduler LREScheduler) error {
	url := fmt.Sprintf("%s/tests/%d/scheduler", c.projectURL(), testID)
	return c.doJSON(http.MethodPut, url, scheduler, nil)
}

// LREScript represents a script in LRE PC.
type LREScript struct {
	Name string `json:"Name"`
	ID   int    `json:"ID"`
}

// ListScripts returns all scripts in the project.
func (c *LREClient) ListScripts() ([]LREScript, error) {
	var scripts []LREScript
	err := c.doJSON(http.MethodGet, c.projectURL()+"/scripts", nil, &scripts)
	return scripts, err
}
