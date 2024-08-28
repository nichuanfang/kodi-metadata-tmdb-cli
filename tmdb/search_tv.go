package tmdb

import (
	"encoding/json"
	"errors"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"sort"
	"strconv"
	"strings"
)

type SearchTvResponse struct {
	Page         int              `json:"page"`
	TotalResults int              `json:"total_results"`
	TotalPages   int              `json:"total_pages"`
	Results      []*SearchResults `json:"results"`
}

type SearchResults struct {
	Id               int      `json:"id"`
	PosterPath       string   `json:"poster_path"`
	Popularity       float32  `json:"popularity"`
	BackdropPath     string   `json:"backdrop_path"`
	VoteAverage      float32  `json:"vote_average"`
	Overview         string   `json:"overview"`
	FirstAirDate     string   `json:"first_air_date"`
	OriginCountry    []string `json:"origin_country"`
	GenreIds         []int    `json:"genre_ids"`
	OriginalLanguage string   `json:"original_language"`
	VoteCount        int      `json:"vote_count"`
	Name             string   `json:"name"`
	OriginalName     string   `json:"original_name"`
}

type Response struct {
	Success       bool   `json:"success"`
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// SearchTvResultsSortWrapper 自定义排序
type SearchTvResultsSortWrapper struct {
	results []*SearchResults
	by      func(l, r *SearchResults) bool
}

func (rw SearchTvResultsSortWrapper) Len() int {
	return len(rw.results)
}
func (rw SearchTvResultsSortWrapper) Swap(i, j int) {
	rw.results[i], rw.results[j] = rw.results[j], rw.results[i]
}
func (rw SearchTvResultsSortWrapper) Less(i, j int) bool {
	return rw.by(rw.results[i], rw.results[j])
}

func swap(s []*SearchResults, i, j int) {
	s[i], s[j] = s[j], s[i]
}

// SortResults 按流行度排序
// TODO 是否有点太粗暴了，考虑多维度：内容完整性、年份、中英文等
// func (d SearchTvResponse) SortResults(year string) {
// 	// 考虑年份 年份对的优先级拉高
// 	sort.Sort(SearchTvResultsSortWrapper{d.Results, func(l, r *SearchResults) bool {
// 		return l.Popularity > r.Popularity
// 	}})
// 	for i := 0; i < len(d.Results); i++ {
// 		firstAirDate := d.Results[i].FirstAirDate
// 		if strings.Split(firstAirDate, "-")[0] == year {
// 			swap(d.Results, 0, i)
// 			break
// 		}
// 	}
// }

// SortResults sorts the TV search results based on multiple criteria
func (resp *SearchTvResponse) SortResults(chsTitle, engTitle, year string) {
	sort.SliceStable(resp.Results, func(i, j int) bool {
		// Check for content completeness
		completeI := isComplete(resp.Results[i])
		completeJ := isComplete(resp.Results[j])
		if completeI != completeJ {
			return completeI
		}

		// If year is provided, prioritize results from that year
		if year != "" {
			yearInt, _ := strconv.Atoi(year)
			yearI, _ := strconv.Atoi(resp.Results[i].FirstAirDate[:4])
			yearJ, _ := strconv.Atoi(resp.Results[j].FirstAirDate[:4])
			if yearI == yearInt && yearJ != yearInt {
				return true
			}
			if yearI != yearInt && yearJ == yearInt {
				return false
			}
		}

		// Check for matching Chinese or English title
		matchTitleI := matchesTitle(resp.Results[i], chsTitle, engTitle)
		matchTitleJ := matchesTitle(resp.Results[j], chsTitle, engTitle)
		if matchTitleI != matchTitleJ {
			return matchTitleI
		}

		// Sort primarily by vote_average, then by popularity
		if resp.Results[i].VoteAverage != resp.Results[j].VoteAverage {
			return resp.Results[i].VoteAverage > resp.Results[j].VoteAverage
		}
		return resp.Results[i].Popularity > resp.Results[j].Popularity
	})
}

// isComplete checks if the result has complete information
func isComplete(result *SearchResults) bool {
	return result.PosterPath != "" && result.BackdropPath != "" && result.Overview != ""
}

// matchesTitle checks if the result matches the given Chinese or English title
func matchesTitle(result *SearchResults, chsTitle, engTitle string) bool {
	return strings.Contains(result.Name, chsTitle) || strings.Contains(result.OriginalName, engTitle)
}

// SearchShows 搜索tmdb
func (t *tmdb) SearchShows(chsTitle, engTitle string, year int) (*SearchResults, error) {
	utils.Logger.InfoF("search: %s or %s %d from tmdb", chsTitle, engTitle, year)

	strYear := strconv.Itoa(year)
	searchComb := make([]map[string]string, 0)

	if chsTitle != "" {
		if year > 0 {
			searchComb = append(searchComb, map[string]string{
				"query":         chsTitle,
				"page":          "1",
				"include_adult": "true",
				"year":          strYear,
			})
		}
		searchComb = append(searchComb, map[string]string{
			"query":         chsTitle,
			"page":          "1",
			"include_adult": "true",
		})
	}

	if engTitle != "" {
		if year > 0 {
			searchComb = append(searchComb, map[string]string{
				"query":         engTitle,
				"page":          "1",
				"include_adult": "true",
				"year":          strYear,
			})
		}
		searchComb = append(searchComb, map[string]string{
			"query":         engTitle,
			"page":          "1",
			"include_adult": "true",
		})
	}

	if len(searchComb) == 0 {
		return nil, errors.New("title empty")
	}

	tvResp := &SearchTvResponse{}
	for _, req := range searchComb {
		body, err := t.request(ApiSearchTv, req)
		if err != nil {
			utils.Logger.ErrorF("read tmdb response err: %v", err)
			continue
		}

		err = json.Unmarshal(body, tvResp)
		if err != nil {
			utils.Logger.ErrorF("parse tmdb response err: %v", err)
			continue
		}

		if len(tvResp.Results) > 0 {
			tvResp.SortResults(chsTitle, engTitle, strYear)
			utils.Logger.InfoF("search tv: %s %d result count: %d, use: %v", chsTitle, year, len(tvResp.Results), tvResp.Results[0])
			return tvResp.Results[0], nil
		}
	}

	return nil, errors.New("search tv not found")
}
