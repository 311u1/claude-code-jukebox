package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bytes"
	"io"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Status struct {
	Stopped     bool   `json:"stopped"`
	Paused      bool   `json:"paused"`
	Buffering   bool   `json:"buffering"`
	Volume      int    `json:"volume"`
	VolumeSteps      int    `json:"volume_steps"`
	ShuffleContext   bool   `json:"shuffle_context"`
	Track            *Track `json:"track"`
}

type Track struct {
	URI           string   `json:"uri"`
	Name          string   `json:"name"`
	ArtistNames   []string `json:"artist_names"`
	AlbumName     string   `json:"album_name"`
	AlbumCoverURL string   `json:"album_cover_url"`
	Position      int      `json:"position"`
	Duration      int      `json:"duration"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Status() (*Status, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to go-librespot at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var s Status
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) post(path string, payload any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", body)
	if err != nil {
		return fmt.Errorf("cannot connect to go-librespot at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) Play(uri string) error {
	return c.post("/player/play", map[string]string{"uri": uri})
}

func (c *Client) PlayPause() error {
	return c.post("/player/playpause", nil)
}

func (c *Client) Next() error {
	return c.post("/player/next", nil)
}

func (c *Client) Prev() error {
	return c.post("/player/prev", nil)
}

func (c *Client) Volume(vol int) error {
	return c.post("/player/volume", map[string]int{"volume": vol})
}

func (c *Client) Seek(ms int) error {
	return c.post("/player/seek", map[string]int{"position": ms})
}

func (c *Client) Shuffle(on bool) error {
	return c.post("/player/shuffle_context", map[string]bool{"shuffle_context": on})
}

func (c *Client) Queue(uri string) error {
	return c.post("/player/add_to_queue", map[string]string{"uri": uri})
}
