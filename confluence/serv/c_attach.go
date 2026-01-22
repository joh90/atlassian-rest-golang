package serv

import (
	"atlas-rest-golang/confluence/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type AttachService struct{}

func (as AttachService) GetPageAttachments(url string, tok string, pid string) models.AttachmentsResult {
	log.Printf("Getting %s page attachments", pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, pid)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicf("Erroe when performing GET request: %s", err)
	}
	defer resp.Body.Close()
	var attachments models.AttachmentsResult
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &attachments)

	return attachments
}

func (as AttachService) AddAttachment(url string, tok string, pid string, attach string) models.Attachment {
	log.Printf("Adding attachment %s to page %s ", attach, pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, pid)

	file, err := os.Open(attach)
	if err != nil {
		log.Printf("Error opening file: %s", err)
		return models.Attachment{}
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(attach))
	if err != nil {
		log.Printf("Error creating form file: %s", err)
		return models.Attachment{}
	}
	bw, err := io.Copy(part, file)
	if err != nil {
		log.Printf("Error copying file data: %s", err)
		return models.Attachment{}
	}
	log.Printf("Bytes written: %d", bw)

	writer.Close() // Must close before request to finalize multipart body

	req, err := http.NewRequest("POST", reqUrl, body)
	if err != nil {
		log.Printf("Error creating request: %s", err)
		return models.Attachment{}
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("X-Atlassian-Token", "nocheck")

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error performing request: %s", err)
		return models.Attachment{}
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %s", err)
		return models.Attachment{}
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("API error (HTTP %d): %s", resp.StatusCode, string(bts))
		return models.Attachment{}
	}

	// API returns {"results": [...]} array
	var result models.AttachmentsResult
	err = json.Unmarshal(bts, &result)
	if err != nil {
		log.Printf("Error parsing response: %s", err)
		return models.Attachment{}
	}

	if len(result.Results) > 0 {
		r := result.Results[0]
		return models.Attachment{
			ID:     r.ID,
			Type:   r.Type,
			Status: r.Status,
			Title:  r.Title,
		}
	}

	log.Printf("No attachment in response")
	return models.Attachment{}
}

func (as AttachService) DownloadAttachmentById(url string, token string, aid string) models.Attachment {
	// todo
	return models.Attachment{}
}

func (as AttachService) DownloadAttachments(url string, token string, pid string) []models.Attachment {
	log.Printf("Downloading page '%s' attachments", pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, pid)

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
	req.Header.Add("Accept", "application/json")

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error when performing GET request: %s", err)
	}
	defer resp.Body.Close()

	var attaches models.AttachmentsResult
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &attaches)

	downloaded := make([]models.Attachment, 0)

	for _, att := range attaches.Results {
		dLink := fmt.Sprintf("%s%s", url, att.Links.Download)
		response, err := client.Get(dLink)
		if err != nil {
			log.Printf("Error when getting attachment via GET HTTP request. Err: %s", err)
			continue
		}
		bts, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			log.Printf("Error reading attachment data: %s", err)
			continue
		}
		// Sanitize filename to prevent path traversal attacks
		safeName := filepath.Base(att.Title)
		err = os.WriteFile(safeName, bts, 0644)
		if err != nil {
			log.Printf("Error writing file %s: %s", safeName, err)
			continue
		}
		downloaded = append(downloaded, models.Attachment{
			ID:    att.ID,
			Title: safeName,
		})
	}

	return downloaded
}
