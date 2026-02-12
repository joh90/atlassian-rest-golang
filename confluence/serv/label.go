package serv

import (
	"atlas-rest-golang/confluence/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type LabelService struct{}

func (l LabelService) GetPageLabels(url string, tok string, pid string) models.LabelArray {
	log.Printf("Getting %s page labels", pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/label", url, pid)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var lArr models.LabelArray
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &lArr)
	log.Println(lArr)

	return lArr
}

// AddLabels adds labels to a page
// Returns: (labels, httpStatus, errorMessage)
func (l LabelService) AddLabels(url string, tok string, pid string, labels []string) (models.LabelArray, int, string) {
	log.Println("Adding labels to " + pid)

	lbls := make([]models.CreateLabel, 0)
	for _, l := range labels {
		lbObj := models.CreateLabel{Prefix: "global", Name: l}
		lbls = append(lbls, lbObj)
	}

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/label", url, pid)
	lJson, err := json.Marshal(lbls)
	if err != nil {
		return models.LabelArray{}, 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(lJson))
	if err != nil {
		return models.LabelArray{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		return models.LabelArray{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.LabelArray{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.LabelArray{}, resp.StatusCode, string(bts)
	}

	var label models.LabelArray
	err = json.Unmarshal(bts, &label)
	if err != nil {
		return models.LabelArray{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return label, resp.StatusCode, ""
}

//func (l LabelService) CopyLabels(url string, tok string, pid string, tgt string) models.LabelArray {
//
//	log.Printf("Copying %s page labels to %s", pid, tgt)
//
//	labels := l.GetPageLabels(url, tok, pid).Results
//
//	l.AddLabel(url, tok, tgt, labels)
//
//	var label models.LabelArray
//	bts, err := ioutil.ReadAll(resp.Body)
//	err = json.Unmarshal(bts, &label)
//
//	return label
//}

//DELETE /rest/api/content/{id}/label/{label}

func (l LabelService) DeleteLabel(url string, tok string, pid string, lid string) string {

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/label/%s", url, pid, lid)
	req, err := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var rStr string
	bts, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &rStr)

	return rStr
}
