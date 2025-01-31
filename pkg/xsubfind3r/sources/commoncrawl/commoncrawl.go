package commoncrawl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hueristiq/xsubfind3r/pkg/httpclient"
	"github.com/hueristiq/xsubfind3r/pkg/xsubfind3r/sources"
)

type getIndexesResponse []struct {
	ID  string `json:"id"`
	API string `json:"cdx-API"`
}

type getPaginationResponse struct {
	Blocks   uint `json:"blocks"`
	PageSize uint `json:"pageSize"`
	Pages    uint `json:"pages"`
}

type getURLsResponse struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

type Source struct{}

func (source *Source) Run(cfg *sources.Configuration, domain string) <-chan sources.Result {
	results := make(chan sources.Result)

	go func() {
		defer close(results)

		getIndexesReqURL := "https://index.commoncrawl.org/collinfo.json"

		getIndexesRes, err := httpclient.SimpleGet(getIndexesReqURL)
		if err != nil {
			result := sources.Result{
				Type:   sources.ResultError,
				Source: source.Name(),
				Error:  err,
			}

			results <- result

			httpclient.DiscardResponse(getIndexesRes)

			return
		}

		var getIndexesResData getIndexesResponse

		if err = json.NewDecoder(getIndexesRes.Body).Decode(&getIndexesResData); err != nil {
			result := sources.Result{
				Type:   sources.ResultError,
				Source: source.Name(),
				Error:  err,
			}

			results <- result

			getIndexesRes.Body.Close()

			return
		}

		getIndexesRes.Body.Close()

		year := time.Now().Year()
		years := make([]string, 0)
		maxYearsBack := 5

		for i := range maxYearsBack {
			years = append(years, strconv.Itoa(year-i))
		}

		searchIndexes := make(map[string]string)

		for _, year := range years {
			for _, CCIndex := range getIndexesResData {
				if strings.Contains(CCIndex.ID, year) {
					if _, ok := searchIndexes[year]; !ok {
						searchIndexes[year] = CCIndex.API

						break
					}
				}
			}
		}

		for _, CCIndexAPI := range searchIndexes {
			getPaginationReqURL := fmt.Sprintf("%s?url=*.%s/*&output=json&fl=url&showNumPages=true", CCIndexAPI, domain)
			getURLsReqHeaders := map[string]string{
				"Host": "index.commoncrawl.org",
			}

			getPaginationRes, err := httpclient.SimpleGet(getPaginationReqURL)
			if err != nil {
				result := sources.Result{
					Type:   sources.ResultError,
					Source: source.Name(),
					Error:  err,
				}

				results <- result

				httpclient.DiscardResponse(getPaginationRes)

				continue
			}

			var getPaginationData getPaginationResponse

			if err = json.NewDecoder(getPaginationRes.Body).Decode(&getPaginationData); err != nil {
				result := sources.Result{
					Type:   sources.ResultError,
					Source: source.Name(),
					Error:  err,
				}

				results <- result

				getPaginationRes.Body.Close()

				continue
			}

			getPaginationRes.Body.Close()

			if getPaginationData.Pages < 1 {
				continue
			}

			for page := range getPaginationData.Pages {
				getURLsReqURL := fmt.Sprintf("%s?url=*.%s/*&output=json&fl=url&page=%d", CCIndexAPI, domain, page)

				getURLsRes, err := httpclient.Get(getURLsReqURL, "", getURLsReqHeaders)
				if err != nil {
					result := sources.Result{
						Type:   sources.ResultError,
						Source: source.Name(),
						Error:  err,
					}

					results <- result

					httpclient.DiscardResponse(getURLsRes)

					continue
				}

				scanner := bufio.NewScanner(getURLsRes.Body)

				for scanner.Scan() {
					line := scanner.Text()
					if line == "" {
						continue
					}

					var getURLsResData getURLsResponse

					if err = json.Unmarshal(scanner.Bytes(), &getURLsResData); err != nil {
						result := sources.Result{
							Type:   sources.ResultError,
							Source: source.Name(),
							Error:  err,
						}

						results <- result

						continue
					}

					if getURLsResData.Error != "" {
						result := sources.Result{
							Type:   sources.ResultError,
							Source: source.Name(),
							Error:  fmt.Errorf("%s", getURLsResData.Error),
						}

						results <- result

						continue
					}

					subdomains := cfg.Extractor.FindAllString(getURLsResData.URL, -1)

					for _, subdomain := range subdomains {
						result := sources.Result{
							Type:   sources.ResultSubdomain,
							Source: source.Name(),
							Value:  subdomain,
						}

						results <- result
					}
				}

				if err = scanner.Err(); err != nil {
					result := sources.Result{
						Type:   sources.ResultError,
						Source: source.Name(),
						Error:  err,
					}

					results <- result

					getURLsRes.Body.Close()

					continue
				}

				getURLsRes.Body.Close()
			}
		}
	}()

	return results
}

func (source *Source) Name() string {
	return sources.COMMONCRAWL
}
