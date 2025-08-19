package outputs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type HTTPDriver struct {
	log        *logrus.Logger
	url        string
	timeout    time.Duration
	parallel   int
	hecToken   string
	httpClient *http.Client
	sem        chan struct{}
}

func NewHTTP(log *logrus.Logger, url string, timeoutMS, parallel int, hecTokenEnv string) *HTTPDriver {
	tok := ""
	if hecTokenEnv != "" {
		tok = os.Getenv(hecTokenEnv)
	}
	if parallel <= 0 {
		parallel = 4
	}
	return &HTTPDriver{
		log:      log,
		url:      url,
		timeout:  time.Duration(timeoutMS) * time.Millisecond,
		parallel: parallel,
		hecToken: tok,
	}
}

func (d *HTTPDriver) Name() string { return "http" }

func (d *HTTPDriver) Start(ctx context.Context) error {
	if d.url == "" {
		return fmt.Errorf("http output url is empty")
	}
	d.httpClient = &http.Client{
		Timeout: d.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        1024,
			MaxIdleConnsPerHost: 1024,
			IdleConnTimeout:     60 * time.Second,
		},
	}
	d.sem = make(chan struct{}, d.parallel)
	return nil
}
func (d *HTTPDriver) Stop(ctx context.Context) error { return nil }

func (d *HTTPDriver) Write(ctx context.Context, batch []Record) error {
	// Here we send an array of raw events; for Splunk HEC, you might prefer
	// event-per-line {"event":<json>} — uncomment the HEC mode if needed.
	payload := make([]json.RawMessage, 0, len(batch))
	for _, r := range batch {
		payload = append(payload, json.RawMessage(r.Body))
	}
	body, _ := json.Marshal(payload)

	d.sem <- struct{}{}
	defer func() { <-d.sem }()

	req, _ := http.NewRequestWithContext(ctx, "POST", d.url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if d.hecToken != "" {
		req.Header.Set("Authorization", "Splunk "+d.hecToken)
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http sink status %d", resp.StatusCode)
	}
	return nil
}

/*
 // Splunk HEC line-by-line variant:
 for _, r := range batch {
   hec := map[string]any{"event": json.RawMessage(r.Body)}
   b, _ := json.Marshal(hec)
   // POST b
 }
*/
