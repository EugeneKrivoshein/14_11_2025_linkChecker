package util

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Now() time.Time { return time.Now() }

func normalize(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)

	hasScheme := strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")

	mainURL := raw
	if !hasScheme {
		mainURL = "https://" + raw
	}

	parsed, err := url.Parse(mainURL)
	if err != nil {
		return mainURL, []string{mainURL}
	}

	host := parsed.Hostname()

	if strings.Count(host, ".") > 1 {
		return mainURL, []string{mainURL}
	}
	variants := []string{
		mainURL,
	}
//на случай таких url https://www.wikipedia.org/
	if !strings.HasPrefix(host, "www.") {
		withWWW := strings.Replace(mainURL, "://", "://www.", 1)
		variants = append(variants, withWWW)
	}

	return mainURL, variants
}

var CheckURL = func(raw string) (bool, string) {
	_, candidates := normalize(raw)

	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			TLSHandshakeTimeout: 3 * time.Second,
		},
	}
	for _, u := range candidates {
		req, _ := http.NewRequest("HEAD", u, nil)
		req.Header.Set("User-Agent",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return true, "ok"
			}
			continue
		}
		req2, _ := http.NewRequest("GET", u, nil)
		req2.Header.Set("User-Agent",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")

		resp2, err2 := client.Do(req2)
		if err2 == nil {
			resp2.Body.Close()
			if resp2.StatusCode >= 200 && resp2.StatusCode < 400 {
				return true, "ok"
			}
			continue
		}

		t := strings.ToLower(err2.Error())
		if strings.Contains(t, "no such host") {
			continue
		}
	}

	return false, "unreachable"
}

