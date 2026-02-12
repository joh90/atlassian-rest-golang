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

// AddAttachment uploads an attachment to a page
// Returns: (attachment, httpStatus, errorMessage)
func (as AttachService) AddAttachment(url string, tok string, pid string, attach string) (models.Attachment, int, string) {
	log.Printf("Adding attachment %s to page %s ", attach, pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, pid)

	file, err := os.Open(attach)
	if err != nil {
		return models.Attachment{}, 0, fmt.Sprintf("Error opening file: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(attach))
	if err != nil {
		return models.Attachment{}, 0, fmt.Sprintf("Error creating form file: %v", err)
	}
	bw, err := io.Copy(part, file)
	if err != nil {
		return models.Attachment{}, 0, fmt.Sprintf("Error copying file data: %v", err)
	}
	log.Printf("Bytes written: %d", bw)

	writer.Close() // Must close before request to finalize multipart body

	req, err := http.NewRequest("POST", reqUrl, body)
	if err != nil {
		return models.Attachment{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("X-Atlassian-Token", "nocheck")

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		return models.Attachment{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Attachment{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Attachment{}, resp.StatusCode, string(bts)
	}

	// API returns {"results": [...]} array
	var result models.AttachmentsResult
	err = json.Unmarshal(bts, &result)
	if err != nil {
		return models.Attachment{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	if len(result.Results) > 0 {
		r := result.Results[0]
		return models.Attachment{
			ID:     r.ID,
			Type:   r.Type,
			Status: r.Status,
			Title:  r.Title,
		}, resp.StatusCode, ""
	}

	return models.Attachment{}, resp.StatusCode, "No attachment in response"
}

func (as AttachService) DownloadAttachmentById(url string, token string, aid string) models.Attachment {
	// todo
	return models.Attachment{}
}

// DownloadAttachments downloads all attachments from a page
// Returns: (attachments, httpStatus, errorMessage)
func (as AttachService) DownloadAttachments(url string, token string, pid string) ([]models.Attachment, int, string) {
	log.Printf("Downloading page '%s' attachments", pid)
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, pid)

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
	req.Header.Add("Accept", "application/json")

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, string(bts)
	}

	var attaches models.AttachmentsResult
	err = json.Unmarshal(bts, &attaches)
	if err != nil {
		return nil, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

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

	return downloaded, resp.StatusCode, ""
}
