package urlscan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hueristiq/xsubfind3r/pkg/xsubfind3r/httpclient"
	"github.com/hueristiq/xsubfind3r/pkg/xsubfind3r/sources"
	"github.com/valyala/fasthttp"
)

type searchResponse struct {
	Results []struct {
		Page struct {
			Domain   string `json:"domain"`
			MimeType string `json:"mimeType"`
			URL      string `json:"url"`
			Status   string `json:"status"`
		} `json:"page"`
		Sort []interface{} `json:"sort"`
	} `json:"results"`
	Status  int  `json:"status"`
	Total   int  `json:"total"`
	Took    int  `json:"took"`
	HasMore bool `json:"has_more"`
}

type Source struct{}

func (source *Source) Run(config *sources.Configuration, domain string) (subdomainsChannel chan sources.Subdomain) {
	subdomainsChannel = make(chan sources.Subdomain)

	go func() {
		defer close(subdomainsChannel)

		var err error

		var key string

		key, err = sources.PickRandom(config.Keys.URLScan)
		if err != nil {
			return
		}

		searchReqHeaders := map[string]string{
			"Content-Type": "application/json",
		}

		if key != "" {
			searchReqHeaders["API-Key"] = key
		}

		var searchAfter []interface{}

		for {
			after := ""

			if searchAfter != nil {
				searchAfterJSON, _ := json.Marshal(searchAfter)
				after = "&search_after=" + string(searchAfterJSON)
			}

			searchReqURL := fmt.Sprintf("https://urlscan.io/api/v1/search/?q=domain:%s&size=100", domain) + after

			var searchRes *fasthttp.Response

			searchRes, err = httpclient.Get(searchReqURL, "", searchReqHeaders)
			if err != nil {
				return
			}

			var searchResData searchResponse

			if err = json.Unmarshal(searchRes.Body(), &searchResData); err != nil {
				return
			}

			if searchResData.Status == 429 {
				break
			}

			for _, result := range searchResData.Results {
				if !strings.HasSuffix(result.Page.Domain, "."+domain) {
					continue
				}

				subdomainsChannel <- sources.Subdomain{Source: source.Name(), Value: result.Page.Domain}
			}

			if !searchResData.HasMore {
				break
			}

			lastResult := searchResData.Results[len(searchResData.Results)-1]
			searchAfter = lastResult.Sort
		}
	}()

	return
}

func (source *Source) Name() string {
	return "urlscan"
}
