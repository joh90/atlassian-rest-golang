package main

import (
	"atlas-rest-golang/confluence/models"
	"atlas-rest-golang/confluence/serv"
	"atlas-rest-golang/jira"
	token "atlas-rest-golang/srv"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {
	runtime.GOMAXPROCS(50)
	start := time.Now()

	argsWithoutProg := os.Args[1:]
	//args := flag.Args()
	//flag.Parse()

	var instanceType string
	var action string

	// confluence
	var pageId string
	var pageTitle string
	var spaceKey string
	var parent string
	var body string
	var labels string
	var find string
	var replace string
	var cql string
	var limit int = 25 // default limit for search/list

	// jira
	var projKey string
	var projId string
	var issueKey string

	//misc
	var file string

	// create issue data
	var summary string
	var description string
	var issTypeId string // 10006
	//var dueDate string   // "2023-04-10"
	var issueLabels []string
	var assignee string
	var reporter string
	//var priorityName string // normal: id=3

	if len(argsWithoutProg) == 0 {
		log.Println("Please specify necessary arguments!")
	}

	for a := 0; a < len(argsWithoutProg); a++ {
		if argsWithoutProg[a] == "--type" {
			instanceType = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--action" {
			action = argsWithoutProg[a+1]
		}

		// confluence
		if argsWithoutProg[a] == "--id" || argsWithoutProg[a] == "--pageId" {
			pageId = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--space" || argsWithoutProg[a] == "--spaceKey" {
			spaceKey = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--title" || argsWithoutProg[a] == "--pageTitle" {
			pageTitle = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--parent" {
			parent = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--body" {
			body = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--file" {
			file = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--labels" {
			labels = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--find" {
			find = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--replace" {
			replace = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--cql" {
			cql = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--limit" {
			fmt.Sscanf(argsWithoutProg[a+1], "%d", &limit)
		}

		// jira
		if argsWithoutProg[a] == "--key" {
			projKey = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--summary" {
			summary = argsWithoutProg[a+1]
		}
		//if argsWithoutProg[a] == "--priority" {
		//	priorityName = argsWithoutProg[a+1]
		//}
		if argsWithoutProg[a] == "--desc" {
			description = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "project" {
			projId = argsWithoutProg[a+1]
		}
	}

	log.Printf(">> \u001b[33m Args are:\u001B[0m %v", argsWithoutProg)

	tokService := token.TokenService{}
	url := os.Getenv("ATLAS_URL")
	//if instanceType == "confluence" {
	//	url += "/wiki"
	//}
	pass := os.Getenv("ATLAS_PASS")

	anmaToken := tokService.GetToken(os.Getenv("ATLAS_USER"), pass)

	// Confluence instance
	switch instanceType {
	case "confluence":
		pageService := serv.PageService{}
		ss := serv.SpaceService{}
		ls := serv.LabelService{}
		as := serv.AttachService{}
		// find space's home page
		if parent == "@home" {
			space := ss.GetSpace(url, anmaToken, spaceKey)
			parent = space.Homepage.Id
		}
		switch action {
		case "getPage":
			if pageId != "" {
				page := pageService.GetPage(url, anmaToken, pageId)
				printPage(page)
			} else {
				page := pageService.GetPageTitleKey(url, anmaToken, spaceKey, pageTitle)
				printPage(page)
			}
		case "getSpace":
			space := ss.GetSpace(url, anmaToken, spaceKey)
			fmt.Println(space)
		case "createPage":
			created := pageService.CreateContent(url, anmaToken, "page", spaceKey, parent, pageTitle, body)
			if labels != "" {
				ls.AddLabels(url, anmaToken, created.Id, strings.Split(labels, ","))
			}
			printPage(created)
		case "addAttach":
			added := as.AddAttachment(url, anmaToken, pageId, file)
			if added.ID != "" {
				log.Printf("Successfully added attachment '%s' (ID: %s) to page %s", added.Title, added.ID, pageId)
			} else {
				log.Printf("Failed to add attachment to page %s", pageId)
			}
			log.Println(file)
		case "downloadAttachments":
			downloaded := as.DownloadAttachments(url, anmaToken, pageId)
			log.Println(downloaded)
		case "addLabel":
			page := pageService.GetPage(url, anmaToken, pageId)
			if labels != "" {
				ls.AddLabels(url, anmaToken, page.Id, strings.Split(labels, ","))
			}
			log.Printf("Added labels '%s' to page '%s'", labels, page.Id)
		case "addComment":
			pageService.AddFooterCommentToPage(url, anmaToken, pageId, body)
		case "updatePage":
			// Find and replace text in a page
			if pageId == "" {
				log.Println("Error: --id is required for updatePage")
				return
			}
			if find == "" {
				log.Println("Error: --find is required for updatePage")
				return
			}
			updated := pageService.UpdatePage(url, anmaToken, pageId, find, replace)
			printPage(updated)
		case "setPageBody":
			// Set the entire body of a page
			if pageId == "" {
				log.Println("Error: --id is required for setPageBody")
				return
			}
			if body == "" {
				log.Println("Error: --body is required for setPageBody (use empty quotes to clear page)")
				return
			}
			updated := pageService.SetPageBody(url, anmaToken, pageId, body, pageTitle)
			printPage(updated)
		case "search":
			// Search using CQL
			if cql == "" {
				log.Println("Error: --cql is required for search")
				return
			}
			results := pageService.SearchCQL(url, anmaToken, cql, limit)
			printSearchResults(results)
		case "listPages":
			// List all pages in a space
			if spaceKey == "" {
				log.Println("Error: --space is required for listPages")
				return
			}
			pages := pageService.GetSpacePages(url, anmaToken, spaceKey)
			printSearchResults(pages)
		case "deletePage":
			// Delete a page (permanent - use archivePage instead if possible)
			if pageId == "" {
				log.Println("Error: --id is required for deletePage")
				return
			}
			success, response := pageService.DeletePage(url, anmaToken, pageId)
			if success {
				log.Printf("Deleted page %s successfully (%s)", pageId, response)
			} else {
				log.Printf("Failed to delete page %s: %s", pageId, response)
			}
		case "archivePage":
			// Archive a page (safer - can be restored)
			if pageId == "" {
				log.Println("Error: --id is required for archivePage")
				return
			}
			success, response := pageService.ArchivePage(url, anmaToken, pageId)
			if success {
				log.Printf("Archived page %s successfully", pageId)
			} else {
				log.Printf("Failed to archive page %s: %s", pageId, response)
			}
		}

	// Jira instance
	case "jira":
		is := jira.IssueService{}
		ps := jira.ProjectService{}
		switch action {
		case "getIssue":
			issue := is.GetIssue(url, anmaToken, issueKey)
			fmt.Println(issue)
		case "getProject":
			proj := ps.GetProject(url, anmaToken, projKey)
			fmt.Println(proj)
		case "createIssue":
			created := is.CreateIssue(url, anmaToken, &jira.CreateIssue{Fields: jira.CreateFields{
				Project:     jira.CreateIssueProject{Id: projId},
				Summary:     summary,
				Issuetype:   jira.CIIssuetype{Id: issTypeId},
				Assignee:    jira.Assignee{Name: assignee},
				Reporter:    jira.Reporter{Name: reporter},
				Labels:      issueLabels,
				Description: description,
			}})
			fmt.Println(created)

		}

	}
	log.Println(file) // todo

	// == END
	fmt.Printf("Operations took '%f' sec", time.Now().Sub(start).Seconds())
}

func printPage(page models.Content) {
	log.Println("============ Content =============")
	log.Printf("\nType: %s\nTitle: %s\nSpace: %s\n \u001b[33mBody: %s\u001b[0m]",
		page.Type, page.Title, page.Space.Name, page.Body.Storage.Value)
}

func printSearchResults(results models.ContentArray) {
	log.Println("============ Search Results =============")
	log.Printf("Found %d results\n", len(results.Results))
	for i, page := range results.Results {
		log.Printf("%d. [%s] %s (ID: %s, Space: %s)\n",
			i+1, page.Type, page.Title, page.Id, page.Space.Key)
	}
}
