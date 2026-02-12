package serv

import (
	"atlas-rest-golang/confluence/models"
	token "atlas-rest-golang/srv"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type PageService struct{}

var (
	labServ = LabelService{}
	//wg      sync.WaitGroup
)

//func basicAuth(username, password string) string {
//	auth := username + ":" + password
//	return base64.StdEncoding.EncodeToString([]byte(auth))
//}

// Cloud Rest API V2
// https://developer.atlassian.com/cloud/confluence/rest/v2/api-group-page/#api-pages-post

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	locUser, _ := os.LookupEnv("ATLAS_USER")
	locPass, _ := os.LookupEnv("ATLAS_PASS")
	tokServ := token.TokenService{}
	tok := tokServ.GetToken(locUser, locPass)
	req.Header.Add("Authorization", "Basic "+tok)
	return nil
}

// GetPageTitleKey retrieves a page by space key and title
// Returns: (content, httpStatus, errorMessage)
func (ps PageService) GetPageTitleKey(url string, tok string, space string, title string) (models.Content, int, string) {
	client := myClient()
	expand := "expand=space,body.storage,history,version"

	encodedTitle := neturl.QueryEscape(title)
	reqUrl := fmt.Sprintf("%s/rest/api/content?spaceKey=%s&title=%s&%s", url, space, encodedTitle, expand)
	log.Println("GET REQ URL is " + reqUrl)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, resp.StatusCode, string(bts)
	}

	var results models.ContentArray
	err = json.Unmarshal(bts, &results)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	if len(results.Results) > 0 {
		return results.Results[0], resp.StatusCode, ""
	}
	return models.Content{}, resp.StatusCode, ""
}

// GetPage retrieves a page by ID
// Returns: (content, httpStatus, errorMessage)
func (ps PageService) GetPage(url string, tok string, id string) (models.Content, int, string) {
	client := myClient()
	expand := "expand=space,body.storage,history,version,metadata.labels"

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s?%s", url, id, expand)
	log.Println("GET REQ URL is " + reqUrl)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content, resp.StatusCode, ""
}

func (s PageService) GetChildren(url string, tok string, id string) models.ContentArray {
	expand := "expand=space,body.storage,history,version"

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/page?%s", url, id, expand)

	req, err := http.NewRequest("GET", reqUrl, nil)
	//defer func(Body io.ReadCloser) {
	//	err := Body.Close()
	//	if err != nil {
	//		log.Panicln(err)
	//	}
	//}(req.Body)
	//req.SetBasicAuth("admin", "admin")
	//resp, err := http.Get(reqUrl)
	req.Header.Add("Authorization", "Basic "+tok)
	resp, err := myClient().Do(req)
	fmt.Printf("Response code for GET_PAGE is %d", resp.StatusCode)
	defer resp.Body.Close()
	if err != nil {
		log.Panicln(err)
	}
	var cnArray models.ContentArray
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &cnArray)

	return cnArray

}

func (s PageService) GetDescendants(url string, tok string, id string, lim int) models.ContentArray {
	//expand := "?expand=body.storage,history,version"

	reqUrl := fmt.Sprintf("%s/rest/api/content/search?cql=ancestor=%s&limit=%d", url, id, lim)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicf("Error performing request GET_DESCENDANTS: %v", err)
	}

	// close request's body
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Panicf("Error closing the response body: %v", err)
		}
	}(resp.Body)

	var cnArray models.ContentArray
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &cnArray)

	return cnArray
}

// CreateContent creates a new page or content
// Returns: (content, httpStatus, errorMessage)
func (ps PageService) CreateContent(url string, tok string, ctype string, key string, parent string,
	title string, body string) (models.Content, int, string) {

	reqUrl := fmt.Sprintf("%s/rest/api/content", url)
	ancestors := []models.Ancestor{{Id: parent}} // parent
	contentBody := models.CreatePage{
		Type:  ctype,
		Title: title,
		CreatePageSpace: models.CreatePageSpace{
			Key: key,
		}, Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: body},
		},
		Ancestors: ancestors,
	}
	mrsCtn, err := json.Marshal(contentBody)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(mrsCtn))
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	resp, err := myClient().Do(req)
	if err != nil {
		return models.Content{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return models.Content{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content, resp.StatusCode, ""
}

func (s PageService) CreateContentAsync(wg *sync.WaitGroup, url string, tok string,
	ctype string, key string, parent string, title string, bd string) models.Content {
	//wg.Add(1)
	defer wg.Done()
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}
	reqUrl := fmt.Sprintf("%s/rest/api/content", url)
	ancts := []models.Ancestor{{Id: parent}} // parent
	cntb := models.CreatePage{
		Type:  ctype,
		Title: title,
		CreatePageSpace: models.CreatePageSpace{
			Key: key,
		}, Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: bd},
		},
		Ancestors: ancts,
	}
	mrsCtn, err2 := json.Marshal(cntb)
	if err2 != nil {
		log.Panicln(err2)
	}
	//fmt.Println(string(mrsCtn))

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(mrsCtn))

	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	var content models.Content

	bts, err := io.ReadAll(resp.Body)

	err = json.Unmarshal(bts, &content)

	return content
}

func (ps PageService) PageContains(url string, tok string, id string, find string) bool {
	page, _, _ := ps.GetPage(url, tok, id)
	return strings.Contains(page.Body.Storage.Value, find)
}

// GetSpacePages lists all pages in a space
// Returns: (results, httpStatus, errorMessage)
func (s PageService) GetSpacePages(url string, tok string, key string) (models.ContentArray, int, string) {
	expand := "expand=body.storage,history,version,space"
	reqUrl := fmt.Sprintf("%s/rest/api/content?type=page&spaceKey=%s&%s&limit=300", url, key, expand)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return models.ContentArray{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)

	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		return models.ContentArray{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ContentArray{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.ContentArray{}, resp.StatusCode, string(bts)
	}

	var cnArray models.ContentArray
	err = json.Unmarshal(bts, &cnArray)
	if err != nil {
		return models.ContentArray{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return cnArray, resp.StatusCode, ""
}
func (ps PageService) GetSpacePagesByLabel(url string, tok string, key string, lb string) models.ContentArray { // todo
	//?cql=space+%3D+"DEV"+and+label+%3D+"aa"
	reqUrl := fmt.Sprintf("%s/rest/api/search?cql=space=\"%s\"+and+label=\"%s\"", url, key, lb)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var cnArray models.ContentArray
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &cnArray)

	return cnArray
}

func (s PageService) GetSpaceBlogs(url string, tok string, key string) models.ContentArray {
	reqUrl := fmt.Sprintf("%s/rest/api/content?type=blogpost&spaceKey=%s&limit=300", url, key) //limit=300
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	client := myClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var cnArray models.ContentArray
	bts, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &cnArray)

	return cnArray

}

func (s PageService) DeletePageLabels(url string, tok string, id string, labels []string) string {
	if len(labels) > 0 {
		for _, lab := range labels {
			reqUrl := fmt.Sprintf("%s/rest/api/content/%s/label/%s", url, id, lab) //limit=300
			req, err := http.NewRequest("DELETE", reqUrl, nil)
			req.Header.Add("Authorization", "Basic "+tok)
			client := myClient()
			resp, err := client.Do(req)
			if err != nil {
				log.Panicln(err)
			}
			defer resp.Body.Close()
			fmt.Println(resp)
		}
		return "labels deleted "
	}
	return "no labels provided"
}

// DeletePage deletes a page permanently (returns success bool and status message)
// Note: Confluence returns HTTP 204 No Content on success (empty body)
func (s PageService) DeletePage(baseUrl string, tok string, id string) (bool, string) {
	log.Printf("Deleting page %s", id)
	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s", baseUrl, id)

	req, err := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Read response body (may be empty on success)
	bts, _ := io.ReadAll(resp.Body)
	responseStr := string(bts)

	// HTTP 204 No Content = success (Confluence doesn't return body on delete)
	// HTTP 200 OK = also success
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return true, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	// Error case - return response body which contains error details
	return false, responseStr
}

// ArchivePage archives a page (safer than delete - can be restored)
func (s PageService) ArchivePage(baseUrl string, tok string, id string) (bool, string) {
	log.Printf("Archiving page %s", id)
	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content/archive", baseUrl)

	// Request body: {"pages": [{"id": 123456789}]}
	type ArchivePage struct {
		Id int64 `json:"id"`
	}
	type ArchiveRequest struct {
		Pages []ArchivePage `json:"pages"`
	}

	// Convert string ID to int64
	var pageId int64
	fmt.Sscanf(id, "%d", &pageId)

	archiveReq := ArchiveRequest{
		Pages: []ArchivePage{{Id: pageId}},
	}

	reqBody, err := json.Marshal(archiveReq)
	if err != nil {
		log.Panicln(err)
	}
	log.Printf("Archive request body: %s", string(reqBody))

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(reqBody))
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	bts, _ := io.ReadAll(resp.Body)
	responseStr := string(bts)
	fmt.Println(responseStr)

	// Check if successful (returns task ID on success)
	if resp.StatusCode == 200 || resp.StatusCode == 202 {
		return true, responseStr
	}
	return false, responseStr
}

func (s PageService) ScrollTemplates(url string, tok string, key string) []string {
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	reqUrl := fmt.Sprintf("%s/plugins/servlet/scroll-office/api/templates?spaceKey=%s", url, key)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	tms := make([]string, 0)
	bts, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &tms)

	return tms
}

func (s PageService) CopyPage(wg *sync.WaitGroup, url string, tok string, pid string, tid string,
	copyLabs bool, copyAtt bool, copyCo bool) models.Content {
	// todo - copyLabels, copyComments, copyAttaches

	log.Println("Copying page " + pid)

	orPage, _, _ := s.GetPage(url, tok, pid)
	//time.Sleep(time.Duration(sservice.ResponseTime) * time.Millisecond) // sleep till getting target parent
	parPage, _, _ := s.GetPage(url, tok, tid)

	/* reqUrl := fmt.Sprintf("%s/rest/api/createdPage", url)
	ancts := []models.Ancestor{{Id: parent}} // parent
	cntb := models.CreateContentAsync{
		//Id:    "",
		Type:  "page",
		Title: orPage.Title,
		Space: models.Space{
			Key: orPage.Space.Key,
		}, Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: orPage.Body.Storage.Value},
		},
		Ancestors: ancts,
	} */
	var ttl string
	if orPage.Space.Key == parPage.Space.Key {
		ttl = "Copy of " + orPage.Title
	} else {
		ttl = orPage.Title
	}

	var createdPage models.Content
	createdPage = s.CreateContentAsync(wg, url, tok, "page", parPage.Space.Key, tid, ttl, orPage.Body.Storage.Value)

	// copy labels
	if copyLabs {
		lArr := labServ.GetPageLabels(url, tok, pid)
		lbls := make([]string, 0)
		for _, l := range lArr.Results {
			lbls = append(lbls, l.Name)
		}
		labServ.AddLabels(url, tok, createdPage.Id, lbls)
	}
	// attachment
	if copyAtt {
		log.Printf("Copying %s page attachments", pid)
		attaches := s.GetPageAttaches(url, tok, pid).Results // todo - can be more than 100 set currently
		for _, att := range attaches {
			s.CopyAttach(url, tok, createdPage.Id, att.Id)
		}
	}
	// comments
	if copyCo {
		// todo
	}

	return createdPage
}

func (s PageService) CopyPageDescs(wg *sync.WaitGroup, url string, tok string, pid string, tgt string, nTitle string,
	copyLabs bool, copyCo bool, copyAtt bool) []models.Content {

	// todo - copyLabels, copyComments, copyAttaches + later 'TargetServer'
	log.Printf("Copying %s page descendants", pid)
	cntList := make([]models.Content, 0)

	//root := s.GetPage(url, tok, pid)
	childs := s.GetChildren(url, tok, pid).Results
	rootCp := s.CopyPage(wg, url, tok, pid, tgt, copyLabs, copyCo, copyAtt)

	log.Printf("ROOT page %s copied as %s", pid, rootCp.Id)

	for _, child := range childs {
		var ttl string
		if nTitle == "" {
			// todo - check current space or different
			if child.Space.Key == rootCp.Space.Key {
				ttl = "Copy of " + child.Title
			} else {
				ttl = child.Title
			}
		} else {
			ttl = nTitle + child.Title
		}
		log.Println("Copying child page " + child.Id + " under " + rootCp.Id)

		// recursion NOT working for GO as in Groovy - use Async ?
		s.CopyPageDescs(wg, url, tok, child.Id, rootCp.Id, ttl, copyLabs, copyCo, copyAtt)
		cpPage := s.CopyPage(wg, url, tok, child.Id, rootCp.Id, copyLabs, copyCo, copyAtt)

		cntList = append(cntList, cpPage)
	}

	return cntList
}

// UpdatePage performs find/replace on a page
// Returns: (content, matchCount, originalVersion, httpStatus, errorMessage)
func (s PageService) UpdatePage(baseUrl string, tok string, pid string, find string, repl string) (models.Content, int, int, int, string) {
	log.Printf("Updating %s page", pid)
	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s", baseUrl, pid)

	// Get current page
	page, status, errMsg := s.GetPage(baseUrl, tok, pid)
	if errMsg != "" {
		return models.Content{}, 0, 0, status, errMsg
	}

	originalVersion := page.Version.Number
	pBody := page.Body.Storage.Value

	// Count matches BEFORE replacing
	matchCount := strings.Count(pBody, find)

	// If no matches, return early without making API call
	if matchCount == 0 {
		return page, 0, originalVersion, 200, ""
	}

	// Perform replacement
	fBody := strings.Replace(pBody, find, repl, -1)

	// Use a minimal struct for PUT request to avoid sending null nested objects
	type EditPageMinimal struct {
		Id      string          `json:"id"`
		Title   string          `json:"title"`
		Type    string          `json:"type"`
		Body    models.Body     `json:"body"`
		Version models.VersionE `json:"version"`
	}

	cntb := EditPageMinimal{
		Id:    page.Id,
		Title: page.Title,
		Type:  "page",
		Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: fBody},
		},
		Version: models.VersionE{Number: originalVersion + 1},
	}

	pageBytes, err := json.Marshal(cntb)
	if err != nil {
		return models.Content{}, matchCount, originalVersion, 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("PUT", reqUrl, bytes.NewReader(pageBytes))
	if err != nil {
		return models.Content{}, matchCount, originalVersion, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return models.Content{}, matchCount, originalVersion, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, matchCount, originalVersion, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, matchCount, originalVersion, resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return models.Content{}, matchCount, originalVersion, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content, matchCount, originalVersion, resp.StatusCode, ""
}

func (s PageService) GetPageAttaches(url string, tok string, pid string) models.ContentArray {

	log.Printf("Getting %s page attachments", pid)
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	expand := "expand=body.storage,history,version"
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment?limit=100&%s", url, pid, expand) // limit=100
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var carr models.ContentArray
	bts, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &carr)
	fmt.Println(string(bts))

	return carr

}

func (s PageService) GetAttach(url string, tok string, aid string) models.Content {

	log.Printf("Getting %s attachment", aid)
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	expand := "expand=body.storage,history,version"
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s?%s", url, aid, expand)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Add("Authorization", "Basic "+tok)
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	var content models.Content
	bts, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &content)
	fmt.Println(string(bts))

	return content

}

func (s PageService) DownloadAttach(url string, tok string, atId string) string {

	log.Printf("Getting %s attachment", atId)
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}
	attach := s.GetAttach(url, tok, atId)

	fileDir, _ := os.Getwd()
	fName := attach.Title

	// download link
	dwLink := attach.Links.Base + attach.Links.Download

	//btb := &bytes.Buffer{} // byte buffer
	//mime.ParseMediaType()

	//reader := multipart.NewReader(btb, "")
	//reader.NextPart()

	r1, _ := http.NewRequest("GET", dwLink, nil)
	r1.Header.Add("Authorization", "Basic "+tok)
	//r1.Header.Add("Content-Type", writer.FormDataContentType())
	r1.Header.Add("X-Atlassian-Token", "nocheck")

	resp, err := client.Do(r1)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	bts, rErr := ioutil.ReadAll(resp.Body)
	if rErr != nil {
		log.Panicln(rErr)
	}
	err = os.WriteFile(fName, bts, fs.ModePerm) // mode 0777
	if err != nil {
		log.Panicln(err)
	}
	filePath := path.Join(fileDir, fName) // file path
	fmt.Println("File path is " + filePath)

	return filePath
}

func (s PageService) CopyAttach(url string, tok string, tpid string, atId string) string {

	log.Println("Adding attach: " + atId + " to page: " + tpid)

	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", url, tpid)

	//req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(mrsCtn))
	// =========

	// get attach
	attach := s.GetAttach(url, tok, atId)

	fileDir, _ := os.Getwd()
	fName := attach.Title
	filePath := path.Join(fileDir, fName) // equals atFilePath
	fmt.Println("File path of attach is: " + filePath)

	atFilePath := s.DownloadAttach(url, tok, atId) // attach.Id = nil

	//os.WriteFile(fName)
	//ioutil.WriteFile(fName)
	// save attach
	fl, _ := os.Open(atFilePath) // atFilePath / fileDir
	defer fl.Close()

	btb := &bytes.Buffer{} // byte buffer
	writer := multipart.NewWriter(btb)
	part, _ := writer.CreateFormFile("file", filepath.Base(fl.Name()))
	io.Copy(part, fl)
	clErr := writer.Close()
	if clErr != nil {
		log.Panicln(clErr)
	}

	r, _ := http.NewRequest("POST", reqUrl, btb)
	cntType := writer.FormDataContentType()
	fmt.Println("Content type is " + cntType)

	r.Header.Add("Authorization", "Basic "+tok)
	r.Header.Add("Content-Type", cntType)
	r.Header.Add("X-Atlassian-Token", "nocheck")
	//client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	//var content models.Content
	bts, err := ioutil.ReadAll(resp.Body)
	//err = json.Unmarshal(bts, &content)
	fmt.Println(string(bts))
	// delete attach
	defer os.Remove(atFilePath)

	return "attachment added " + string(bts)

}

func (ps PageService) GetComment(url string, tok string, cid string) models.Content {
	log.Printf("Getting comment %s", cid)

	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	expand := "expand=body.storage,history,version"

	reqUrl := fmt.Sprintf("%s/rest/api/content/%s?%s", url, cid, expand)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		log.Panicf("Error creating GET request for comment: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		log.Panicf("Error perforing request: %v", err)
	}
	defer resp.Body.Close()

	var content models.Content

	bts, err := io.ReadAll(resp.Body)

	err = json.Unmarshal(bts, &content)

	return content
}

// AddFooterCommentToPage adds a comment to a page
// Returns: (commentId, httpStatus, errorMessage)
func (ps PageService) AddFooterCommentToPage(url string, token string, pageId string, body string) (string, int, string) {
	log.Printf("Adding comment to page '%s'", pageId)

	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content", url)

	// Build request body as struct for proper JSON escaping
	type CommentContainer struct {
		Id   string `json:"id"`
		Type string `json:"type"`
	}
	type CommentBody struct {
		Type      string           `json:"type"`
		Status    string           `json:"status"`
		Container CommentContainer `json:"container"`
		Body      models.Body      `json:"body"`
	}

	commentReq := CommentBody{
		Type:   "comment",
		Status: "current",
		Container: CommentContainer{
			Id:   pageId,
			Type: "page",
		},
		Body: models.Body{
			Storage: models.Storage{
				Representation: "storage",
				Value:          body,
			},
		},
	}

	reqBody, err := json.Marshal(commentReq)
	if err != nil {
		return "", 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(reqBody))
	if err != nil {
		return "", 0, fmt.Sprintf("Error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return "", resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content.Id, resp.StatusCode, ""
}

// ReplyToComment replies to an existing comment
// Returns: (replyCommentId, httpStatus, errorMessage)
func (ps PageService) ReplyToComment(url string, token string, parentCommentId string, body string) (string, int, string) {
	log.Printf("Replying to comment '%s'", parentCommentId)

	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content", url)

	// Build request body - container is the parent comment
	type CommentContainer struct {
		Id   string `json:"id"`
		Type string `json:"type"`
	}
	type CommentBody struct {
		Type      string           `json:"type"`
		Status    string           `json:"status"`
		Container CommentContainer `json:"container"`
		Body      models.Body      `json:"body"`
	}

	commentReq := CommentBody{
		Type:   "comment",
		Status: "current",
		Container: CommentContainer{
			Id:   parentCommentId,
			Type: "comment", // Parent is a comment, not a page
		},
		Body: models.Body{
			Storage: models.Storage{
				Representation: "storage",
				Value:          body,
			},
		},
	}

	reqBody, err := json.Marshal(commentReq)
	if err != nil {
		return "", 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(reqBody))
	if err != nil {
		return "", 0, fmt.Sprintf("Error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return "", resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content.Id, resp.StatusCode, ""
}

// EditComment updates an existing comment's content (full body replacement)
// Returns: (content, originalVersion, httpStatus, errorMessage)
func (ps PageService) EditComment(baseUrl string, tok string, commentId string, newBody string) (models.Content, int, int, string) {
	log.Printf("Editing comment '%s'", commentId)

	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s", baseUrl, commentId)

	// Get current comment to get version number
	comment := ps.GetComment(baseUrl, tok, commentId)
	if comment.Id == "" {
		return models.Content{}, 0, 404, "Comment not found"
	}

	originalVersion := comment.Version.Number

	// Use a minimal struct for PUT request
	type EditCommentMinimal struct {
		Id      string          `json:"id"`
		Type    string          `json:"type"`
		Status  string          `json:"status"`
		Body    models.Body     `json:"body"`
		Version models.VersionE `json:"version"`
	}

	editReq := EditCommentMinimal{
		Id:     comment.Id,
		Type:   "comment",
		Status: "current",
		Body: models.Body{
			Storage: models.Storage{
				Representation: "storage",
				Value:          newBody,
			},
		},
		Version: models.VersionE{Number: originalVersion + 1},
	}

	reqBody, err := json.Marshal(editReq)
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("PUT", reqUrl, bytes.NewReader(reqBody))
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, originalVersion, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, originalVersion, resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return models.Content{}, originalVersion, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content, originalVersion, resp.StatusCode, ""
}

func (p PageService) AddComment(url string, tok string, cid string, pid string) models.Content {
	log.Printf("Copying %s comment to %s page", cid, pid)
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	cmm := p.GetComment(url, tok, cid)
	page, _, _ := p.GetPage(url, tok, pid)

	reqUrl := fmt.Sprintf("%s/rest/api/content", url)
	cntb := models.CreateComment{
		//Id:    "",
		Type:  "comment",
		Title: cmm.Title,
		CreatePageSpace: models.CreatePageSpace{
			Key: cmm.Space.Key,
		}, Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: cmm.Body.Storage.Value},
		},
		Container: page,
	}
	mrsCtn, err2 := json.Marshal(cntb)
	if err2 != nil {
		log.Panicln(err2)
	}
	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(mrsCtn))
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	var content models.Content
	bts, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bts, &content)
	fmt.Println(string(bts))

	return content

}

func myClient() *http.Client {
	return &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}
}

// SetPageBody sets the entire body of a page (replaces all content)
// If newTitle is empty, keeps the existing title
// Returns: (content, originalVersion, httpStatus, errorMessage)
func (s PageService) SetPageBody(baseUrl string, tok string, pid string, newBody string, newTitle string) (models.Content, int, int, string) {
	log.Printf("Setting body for page %s", pid)
	client := myClient()
	reqUrl := fmt.Sprintf("%s/rest/api/content/%s", baseUrl, pid)

	// Get current page to get version number and title
	page, status, errMsg := s.GetPage(baseUrl, tok, pid)
	if errMsg != "" {
		return models.Content{}, 0, status, errMsg
	}

	originalVersion := page.Version.Number
	title := page.Title
	if newTitle != "" {
		title = newTitle
	}

	// Use a minimal struct for PUT request to avoid sending null nested objects
	type EditPageMinimal struct {
		Id      string          `json:"id"`
		Title   string          `json:"title"`
		Type    string          `json:"type"`
		Body    models.Body     `json:"body"`
		Version models.VersionE `json:"version"`
	}

	cntb := EditPageMinimal{
		Id:    page.Id,
		Title: title,
		Type:  "page",
		Body: models.Body{
			Storage: models.Storage{
				Representation: "storage", Value: newBody},
		},
		Version: models.VersionE{Number: originalVersion + 1},
	}

	pageBytes, err := json.Marshal(cntb)
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error marshalling request: %v", err)
	}

	req, err := http.NewRequest("PUT", reqUrl, bytes.NewReader(pageBytes))
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return models.Content{}, originalVersion, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Content{}, originalVersion, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.Content{}, originalVersion, resp.StatusCode, string(bts)
	}

	var content models.Content
	err = json.Unmarshal(bts, &content)
	if err != nil {
		return models.Content{}, originalVersion, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return content, originalVersion, resp.StatusCode, ""
}

// SearchCQL searches Confluence using CQL (Confluence Query Language)
// Returns: (results, httpStatus, errorMessage)
func (s PageService) SearchCQL(baseUrl string, tok string, cql string, limit int) (models.ContentArray, int, string) {
	log.Printf("Searching with CQL: %s (limit: %d)", cql, limit)
	client := myClient()

	// URL encode the CQL query to handle spaces, quotes, and special characters
	encodedCQL := neturl.QueryEscape(cql)
	reqUrl := fmt.Sprintf("%s/rest/api/content/search?cql=%s&limit=%d&expand=space", baseUrl, encodedCQL, limit)
	log.Println("Search URL: " + reqUrl)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return models.ContentArray{}, 0, fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Add("Authorization", "Basic "+tok)

	resp, err := client.Do(req)
	if err != nil {
		return models.ContentArray{}, 0, fmt.Sprintf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ContentArray{}, resp.StatusCode, fmt.Sprintf("Error reading response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.ContentArray{}, resp.StatusCode, string(bts)
	}

	var results models.ContentArray
	err = json.Unmarshal(bts, &results)
	if err != nil {
		return models.ContentArray{}, resp.StatusCode, fmt.Sprintf("Error parsing response: %v", err)
	}

	return results, resp.StatusCode, ""
}
