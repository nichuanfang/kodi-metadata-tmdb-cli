package tmdb

import (
	"encoding/json"
	"errors"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"sort"
	"strconv"
	"strings"
)

func (t *tmdb) SearchMovie(chsTitle, engTitle string, year int) (*SearchMoviesResults, error) {
	utils.Logger.InfoF("search: %s or %s %d from tmdb", chsTitle, engTitle, year)

	strYear := strconv.Itoa(year)
	searchComb := make([]map[string]string, 0)

	if chsTitle != "" {
		// chs + year
		if year > 0 {
			searchComb = append(searchComb, map[string]string{
				"query":         chsTitle,
				"page":          "1",
				"include_adult": "true",
				//"region": "US",
				"year":                 strYear,
				"primary_release_year": strYear,
			})
		}
		// chs
		searchComb = append(searchComb, map[string]string{
			"query":         chsTitle,
			"page":          "1",
			"include_adult": "true",
			//"region": "US",
		})
	}

	if engTitle != "" {
		// eng + year
		if year > 0 {
			searchComb = append(searchComb, map[string]string{
				"query":         engTitle,
				"page":          "1",
				"include_adult": "true",
				//"region": "US",
				"year":                 strYear,
				"primary_release_year": strYear,
			})
		}
		// eng
		searchComb = append(searchComb, map[string]string{
			"query":         engTitle,
			"page":          "1",
			"include_adult": "true",
			//"region": "US",
		})
	}

	if len(searchComb) == 0 {
		return nil, errors.New("title empty")
	}

	moviesResp := &SearchMoviesResponse{}
	for _, req := range searchComb {
		body, err := t.request(ApiSearchMovie, req)
		if err != nil {
			utils.Logger.ErrorF("read tmdb response err: %v", err)
			continue
		}

		err = json.Unmarshal(body, moviesResp)
		if err != nil {
			utils.Logger.ErrorF("parse tmdb response err: %v", err)
			continue
		}

		if len(moviesResp.Results) > 0 {
			moviesResp.SortResults(chsTitle, engTitle, strYear)
			utils.Logger.InfoF("search movies: %s %d result count: %d, use: %v", chsTitle, year, len(moviesResp.Results), moviesResp.Results[0])
			return moviesResp.Results[0], nil
		}
	}

	return nil, errors.New("search movie not found")
}

// SortResults sorts the movie search results based on multiple criteria
func (resp *SearchMoviesResponse) SortResults(chsTitle, engTitle, year string) {
	sort.SliceStable(resp.Results, func(i, j int) bool {
		// Check for content completeness
		completeI := isCompleteMovie(resp.Results[i])
		completeJ := isCompleteMovie(resp.Results[j])
		if completeI != completeJ {
			return completeI
		}

		// If year is provided, prioritize results from that year
		if year != "" {
			yearInt, _ := strconv.Atoi(year)
			yearI, _ := strconv.Atoi(resp.Results[i].ReleaseDate[:4])
			yearJ, _ := strconv.Atoi(resp.Results[j].ReleaseDate[:4])
			if yearI == yearInt && yearJ != yearInt {
				return true
			}
			if yearI != yearInt && yearJ == yearInt {
				return false
			}
		}

		// Check for matching Chinese or English title
		matchTitleI := matchesTitleMovie(resp.Results[i], chsTitle, engTitle)
		matchTitleJ := matchesTitleMovie(resp.Results[j], chsTitle, engTitle)
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

// isCompleteMovie checks if the movie result has complete information
func isCompleteMovie(result *SearchMoviesResults) bool {
	return result.PosterPath != "" && result.BackdropPath != "" && result.Overview != ""
}

// matchesTitleMovie checks if the movie result matches the given Chinese or English title
func matchesTitleMovie(result *SearchMoviesResults, chsTitle, engTitle string) bool {
	return strings.Contains(result.Title, chsTitle) || strings.Contains(result.OriginalTitle, engTitle)
}
