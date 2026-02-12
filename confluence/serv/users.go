package serv

import (
	"atlas-rest-golang/confluence/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type UserService struct{}

// UserSearchResponse represents the response from user search API
type UserSearchResponse struct {
	Results   []UserSearchResult `json:"results"`
	Start     int                `json:"start"`
	Limit     int                `json:"limit"`
	TotalSize int                `json:"totalSize"`
}

// UserSearchResult represents a user search result
type UserSearchResult struct {
	User models.User `json:"user"`
}

// SearchUsers searches for users using CQL query
// Returns: (users, httpStatus, errorMessage)
func (us UserService) SearchUsers(baseUrl string, tok string, cql string, limit int) ([]models.User, int, string) {
	client := myClient()

	encodedCQL := url.QueryEscape(cql)

	reqUrl := fmt.Sprintf("%s/rest/api/search/user?cql=%s&limit=%d", baseUrl, encodedCQL, limit)
	log.Println("GET REQ URL is " + reqUrl)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, 0, fmt.Sprintf("Error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, string(bts)
	}

	var searchResp UserSearchResponse
	err = json.Unmarshal(bts, &searchResp)
	if err != nil {
		return nil, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	// Extract users from search results
	users := make([]models.User, len(searchResp.Results))
	for i, result := range searchResp.Results {
		users[i] = result.User
	}

	return users, resp.StatusCode, ""
}
