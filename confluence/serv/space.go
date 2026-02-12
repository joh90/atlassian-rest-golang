package serv

import (
	"atlas-rest-golang/confluence/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type SpaceService struct{}

// GetSpace retrieves a space by key
// Returns: (space, httpStatus, errorMessage)
func (s SpaceService) GetSpace(url string, tok string, key string) (models.Space, int, string) {
	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/space/%s?expand=homepage,metadata.labels", url, key)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return models.Space{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		return models.Space{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Space{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Space{}, resp.StatusCode, string(bts)
	}

	var space models.Space
	err = json.Unmarshal(bts, &space)
	if err != nil {
		return models.Space{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return space, resp.StatusCode, ""
}

func (s SpaceService) CreateSpace(url string, tok string, key string, name string) models.Space {
	log.Printf("Creating space %s", name)

	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	space := models.CreateSpace{Key: key, Name: name}
	bod, _ := json.Marshal(space)
	reqUrl := fmt.Sprintf("%s/rest/api/space", url)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(bod))
	//req.SetBasicAuth("admin", "admin")
	//resp, err := http.Get(reqUrl)
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	rspb, err2 := io.ReadAll(resp.Body)
	if err2 != nil {
		log.Println(err2)
	}
	log.Printf("Response is %s", string(rspb))
	var crSPace models.Space
	err4 := json.Unmarshal(rspb, crSPace)
	if err != nil {
		log.Panicln(err4)
	}

	return crSPace
}
