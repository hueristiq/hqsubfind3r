package leakix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hueristiq/xsubfind3r/pkg/httpclient"
	"github.com/hueristiq/xsubfind3r/pkg/xsubfind3r/sources"
)

type getSubdomainsResponse struct {
	Subdomain   string    `json:"subdomain"`
	DistinctIps int       `json:"distinct_ips"`
	LastSeen    time.Time `json:"last_seen"`
}

type Source struct{}

func (source *Source) Run(config *sources.Configuration, domain string) <-chan sources.Result {
	results := make(chan sources.Result)

	go func() {
		defer close(results)

		key, err := config.Keys.LeakIX.PickRandom()
		if key == "" || err != nil {
			result := sources.Result{
				Type:   sources.ResultError,
				Source: source.Name(),
				Error:  err,
			}

			results <- result

			return
		}

		getSubdomainsReqHeaders := map[string]string{
			"accept": "application/json",
		}

		if len(config.Keys.Bevigil) > 0 {
			getSubdomainsReqHeaders["api-key"] = key
		}

		getSubdomainsReqURL := fmt.Sprintf("https://leakix.net/api/subdomains/%s", domain)

		var getSubdomainsRes *http.Response

		getSubdomainsRes, err = httpclient.Get(getSubdomainsReqURL, "", getSubdomainsReqHeaders)
		if err != nil {
			result := sources.Result{
				Type:   sources.ResultError,
				Source: source.Name(),
				Error:  err,
			}

			results <- result

			httpclient.DiscardResponse(getSubdomainsRes)

			return
		}

		var getSubdomainsResData []getSubdomainsResponse

		if err = json.NewDecoder(getSubdomainsRes.Body).Decode(&getSubdomainsResData); err != nil {
			result := sources.Result{
				Type:   sources.ResultError,
				Source: source.Name(),
				Error:  err,
			}

			results <- result

			getSubdomainsRes.Body.Close()

			return
		}

		getSubdomainsRes.Body.Close()

		for _, record := range getSubdomainsResData {
			result := sources.Result{
				Type:   sources.ResultSubdomain,
				Source: source.Name(),
				Value:  record.Subdomain,
			}

			results <- result
		}
	}()

	return results
}

func (source *Source) Name() string {
	return sources.LEAKIX
}
