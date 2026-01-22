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
	var limit int = 25    // default limit for search/list
	var commentId string  // for reply/edit comment
	var verbose bool      // re-fetch and print body after write operations

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
		if argsWithoutProg[a] == "--commentId" {
			commentId = argsWithoutProg[a+1]
		}
		if argsWithoutProg[a] == "--verbose" || argsWithoutProg[a] == "-v" {
			verbose = true
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
				page, status, errMsg := pageService.GetPage(url, anmaToken, pageId)
				if errMsg != "" {
					fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
				} else {
					fmt.Println("SUCCESS: Retrieved page")
					fmt.Printf("  ID: %s\n", page.Id)
					fmt.Printf("  Title: %s\n", page.Title)
					fmt.Printf("  Space: %s\n", page.Space.Name)
					fmt.Printf("  Status: %s\n", page.Status)
					fmt.Printf("  Version: %d\n", page.Version.Number)
					if len(page.Metadata.Labels.Results) > 0 {
						var labelNames []string
						for _, label := range page.Metadata.Labels.Results {
							labelNames = append(labelNames, label.Name)
						}
						fmt.Printf("  Labels: %s\n", strings.Join(labelNames, ", "))
					}
					fmt.Println("Body:")
					fmt.Println(page.Body.Storage.Value)
				}
			} else {
				page := pageService.GetPageTitleKey(url, anmaToken, spaceKey, pageTitle)
				if page.Id != "" {
					fmt.Println("SUCCESS: Retrieved page")
					fmt.Printf("  ID: %s\n", page.Id)
					fmt.Printf("  Title: %s\n", page.Title)
					fmt.Println("Body:")
					fmt.Println(page.Body.Storage.Value)
				} else {
					fmt.Println("FAILED: Page not found")
				}
			}
		case "getSpace":
			if spaceKey == "" {
				fmt.Println("FAILED: --space is required for getSpace")
				return
			}
			space := ss.GetSpace(url, anmaToken, spaceKey)
			if space.Key != "" {
				fmt.Println("SUCCESS: Retrieved space")
				fmt.Printf("  Key: %s\n", space.Key)
				fmt.Printf("  Name: %s\n", space.Name)
				fmt.Printf("  Type: %s\n", space.Type)
				fmt.Printf("  Status: %s\n", space.Status)
				if space.Homepage.Id != "" {
					fmt.Printf("  Homepage ID: %s\n", space.Homepage.Id)
					fmt.Printf("  Homepage Title: %s\n", space.Homepage.Title)
				}
			} else {
				fmt.Printf("FAILED: Space '%s' not found\n", spaceKey)
			}
		case "createPage":
			created, status, errMsg := pageService.CreateContent(url, anmaToken, "page", spaceKey, parent, pageTitle, body)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else if created.Id != "" {
				fmt.Println("SUCCESS: Created page")
				fmt.Printf("  ID: %s\n", created.Id)
				fmt.Printf("  Title: %s\n", created.Title)
				fmt.Printf("  Space: %s\n", spaceKey)
				fmt.Printf("  Version: %d\n", created.Version.Number)
				if labels != "" {
					ls.AddLabels(url, anmaToken, created.Id, strings.Split(labels, ","))
					fmt.Printf("  Labels: %s\n", labels)
				}
				if verbose {
					// Re-fetch to show actual stored content
					refetched, _, _ := pageService.GetPage(url, anmaToken, created.Id)
					fmt.Println("Body (stored):")
					fmt.Println(refetched.Body.Storage.Value)
				}
			} else {
				fmt.Println("FAILED: Could not create page")
			}
		case "addAttach":
			if pageId == "" {
				fmt.Println("FAILED: --id is required for addAttach")
				return
			}
			if file == "" {
				fmt.Println("FAILED: --file is required for addAttach")
				return
			}
			added := as.AddAttachment(url, anmaToken, pageId, file)
			if added.ID != "" {
				fmt.Println("SUCCESS: Added attachment")
				fmt.Printf("  Attachment ID: %s\n", added.ID)
				fmt.Printf("  Title: %s\n", added.Title)
				fmt.Printf("  Page ID: %s\n", pageId)
			} else {
				fmt.Printf("FAILED: Could not add attachment to page %s\n", pageId)
			}
		case "downloadAttachments":
			if pageId == "" {
				fmt.Println("FAILED: --id is required for downloadAttachments")
				return
			}
			downloaded := as.DownloadAttachments(url, anmaToken, pageId)
			fmt.Printf("SUCCESS: Downloaded %d attachment(s) to ./\n", len(downloaded))
			for i, att := range downloaded {
				fmt.Printf("  [%d] %s\n", i+1, att.Title)
			}
		case "addLabel":
			page, status, errMsg := pageService.GetPage(url, anmaToken, pageId)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else if labels != "" {
				ls.AddLabels(url, anmaToken, page.Id, strings.Split(labels, ","))
				fmt.Println("SUCCESS: Added labels")
				fmt.Printf("  ID: %s\n", page.Id)
				fmt.Printf("  Labels: %s\n", labels)
			} else {
				fmt.Println("FAILED: No labels specified")
			}
		case "addComment":
			if pageId == "" {
				fmt.Println("FAILED: --id is required for addComment")
				return
			}
			if body == "" {
				fmt.Println("FAILED: --body is required for addComment")
				return
			}
			commentID, status, errMsg := pageService.AddFooterCommentToPage(url, anmaToken, pageId, body)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else {
				fmt.Println("SUCCESS: Added comment")
				fmt.Printf("  Comment ID: %s\n", commentID)
				fmt.Printf("  Page ID: %s\n", pageId)
			}
		case "updatePage":
			// Find and replace text in a page
			if pageId == "" {
				fmt.Println("FAILED: --id is required for updatePage")
				return
			}
			if find == "" {
				fmt.Println("FAILED: --find is required for updatePage")
				return
			}
			updated, matchCount, origVersion, status, errMsg := pageService.UpdatePage(url, anmaToken, pageId, find, replace)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else if matchCount == 0 {
				fmt.Printf("WARNING: No matches found for \"%s\" - page unchanged\n", find)
				fmt.Printf("  ID: %s\n", pageId)
				if verbose {
					// Re-fetch to show current content for debugging
					refetched, _, _ := pageService.GetPage(url, anmaToken, pageId)
					fmt.Println("Body (current):")
					fmt.Println(refetched.Body.Storage.Value)
				}
			} else {
				fmt.Printf("SUCCESS: Replaced %d occurrence(s) of \"%s\"\n", matchCount, find)
				fmt.Printf("  ID: %s\n", updated.Id)
				fmt.Printf("  Version: %d → %d\n", origVersion, updated.Version.Number)
				if verbose {
					// Re-fetch to show actual stored content
					refetched, _, _ := pageService.GetPage(url, anmaToken, updated.Id)
					fmt.Println("Body (stored):")
					fmt.Println(refetched.Body.Storage.Value)
				}
			}
		case "setPageBody":
			// Set the entire body of a page
			if pageId == "" {
				fmt.Println("FAILED: --id is required for setPageBody")
				return
			}
			if body == "" {
				fmt.Println("FAILED: --body is required for setPageBody (use empty quotes to clear page)")
				return
			}
			updated, origVersion, status, errMsg := pageService.SetPageBody(url, anmaToken, pageId, body, pageTitle)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else {
				fmt.Println("SUCCESS: Updated page body")
				fmt.Printf("  ID: %s\n", updated.Id)
				fmt.Printf("  Version: %d → %d\n", origVersion, updated.Version.Number)
				if verbose {
					// Re-fetch to show actual stored content
					refetched, _, _ := pageService.GetPage(url, anmaToken, updated.Id)
					fmt.Println("Body (stored):")
					fmt.Println(refetched.Body.Storage.Value)
				}
			}
		case "search":
			// Search using CQL
			if cql == "" {
				fmt.Println("FAILED: --cql is required for search")
				return
			}
			results := pageService.SearchCQL(url, anmaToken, cql, limit)
			printSearchResults(results, limit)
		case "listPages":
			// List all pages in a space
			if spaceKey == "" {
				fmt.Println("FAILED: --space is required for listPages")
				return
			}
			pages := pageService.GetSpacePages(url, anmaToken, spaceKey)
			printSearchResults(pages, 0)
		case "deletePage":
			// Delete a page (permanent - use archivePage instead if possible)
			if pageId == "" {
				fmt.Println("FAILED: --id is required for deletePage")
				return
			}
			success, response := pageService.DeletePage(url, anmaToken, pageId)
			if success {
				fmt.Println("SUCCESS: Deleted page")
				fmt.Printf("  ID: %s\n", pageId)
				fmt.Printf("  Status: %s\n", response)
			} else {
				fmt.Printf("FAILED: Could not delete page %s - %s\n", pageId, response)
			}
		case "archivePage":
			// Archive a page (safer - can be restored)
			if pageId == "" {
				fmt.Println("FAILED: --id is required for archivePage")
				return
			}
			success, response := pageService.ArchivePage(url, anmaToken, pageId)
			if success {
				fmt.Println("SUCCESS: Archived page")
				fmt.Printf("  ID: %s\n", pageId)
			} else {
				fmt.Printf("FAILED: Could not archive page %s - %s\n", pageId, response)
			}
		case "replyComment":
			// Reply to an existing comment
			if commentId == "" {
				fmt.Println("FAILED: --commentId is required for replyComment")
				return
			}
			if body == "" {
				fmt.Println("FAILED: --body is required for replyComment")
				return
			}
			replyID, status, errMsg := pageService.ReplyToComment(url, anmaToken, commentId, body)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else {
				fmt.Println("SUCCESS: Replied to comment")
				fmt.Printf("  Reply ID: %s\n", replyID)
				fmt.Printf("  Parent Comment ID: %s\n", commentId)
			}
		case "editComment":
			// Edit an existing comment (full body replacement)
			if commentId == "" {
				fmt.Println("FAILED: --commentId is required for editComment")
				return
			}
			if body == "" {
				fmt.Println("FAILED: --body is required for editComment")
				return
			}
			updated, origVersion, status, errMsg := pageService.EditComment(url, anmaToken, commentId, body)
			if errMsg != "" {
				fmt.Printf("FAILED: HTTP %d - %s\n", status, errMsg)
			} else {
				fmt.Println("SUCCESS: Updated comment")
				fmt.Printf("  Comment ID: %s\n", updated.Id)
				fmt.Printf("  Version: %d → %d\n", origVersion, updated.Version.Number)
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
	fmt.Printf("Operations took '%f' sec\n", time.Now().Sub(start).Seconds())
}

func printPage(page models.Content) {
	log.Println("============ Content =============")
	log.Printf("\nType: %s\nTitle: %s\nSpace: %s\n \u001b[33mBody: %s\u001b[0m]",
		page.Type, page.Title, page.Space.Name, page.Body.Storage.Value)
}

func printSearchResults(results models.ContentArray, limit int) {
	if limit > 0 {
		fmt.Printf("SUCCESS: Found %d result(s) (limit: %d)\n", len(results.Results), limit)
	} else {
		fmt.Printf("SUCCESS: Found %d result(s)\n", len(results.Results))
	}
	for i, page := range results.Results {
		fmt.Printf("  [%d] ID: %s | Type: %s | Title: %s | Space: %s\n",
			i+1, page.Id, page.Type, page.Title, page.Space.Key)
	}
}
