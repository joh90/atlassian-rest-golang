# Atlas - Confluence & Jira REST CLI

CLI tool for interacting with Confluence and Jira REST APIs.

Forked from [AndriiMaliuta/atlassian-rest-golang](https://github.com/AndriiMaliuta/atlassian-rest-golang).

## Setup

### Download

Release binaries are in the [`bin/`](bin/) folder, organized by version.

| Version | Date | Highlights |
|---------|------|------------|
| [v0.0.11](bin/v0.0.11/) | 2026-02-12 | searchUsers, error handling overhaul, getPage-by-title fix |
| [v0.0.10](bin/v0.0.10/) | 2026-01-22 | Structured output (SUCCESS/FAILED/WARNING), labels in getPage |
| [v0.0.9.2](bin/v0.0.9.2/) | 2026-01-12 | Attachment upload fix |
| [v0.0.9.1](bin/v0.0.9.1/) | 2025-12-11 | search, listPages, updatePage, archivePage, deletePage |

### Environment Variables

```bash
export ATLAS_URL="https://your-instance.atlassian.net/wiki"
export ATLAS_USER="your_email@example.com"
export ATLAS_PASS="your_api_token"
```

Generate an API token at https://id.atlassian.com/manage-profile/security/api-tokens.

## CLI Usage

### Confluence

#### Read

```bash
# Get page by ID
atlas --type confluence --action getPage --id "854950177"

# Get page by space + title
atlas --type confluence --action getPage --space "TEST" --title "Page Title"

# Get space details
atlas --type confluence --action getSpace --space "TEST"

# List all pages in a space
atlas --type confluence --action listPages --space "TEST"
```

#### Search

```bash
# CQL search
atlas --type confluence --action search --cql 'title~"keyword"' --limit 10
atlas --type confluence --action search --cql 'space="TEST" AND type="page"' --limit 20

# Search users
atlas --type confluence --action searchUsers --cql 'user.fullname~"john"' --limit 10
```

#### Create

```bash
# Create page
atlas --type confluence --action createPage --title "New Page" --space "TEST" \
  --body "<p>Content</p>" --parent "858488858"

# Create page at space root
atlas --type confluence --action createPage --title "New Page" --space "TEST" \
  --body "<p>Content</p>" --parent "@home"

# Create page with labels
atlas --type confluence --action createPage --title "New Page" --space "TEST" \
  --body "<p>Content</p>" --parent "@home" --labels "tag1,tag2"

# Add comment
atlas --type confluence --action addComment --id "123456" --body "<p>Comment text</p>"

# Add labels
atlas --type confluence --action addLabel --id "123456" --labels "tag1,tag2"

# Create space
atlas --type confluence --action createSpace --name "My Space" --category "team"
```

#### Update

```bash
# Find and replace text in page body
atlas --type confluence --action updatePage --id "123456" --find "old text" --replace "new text"

# Set entire page body
atlas --type confluence --action setPageBody --id "123456" --body "<p>New content</p>"

# Set body and rename title
atlas --type confluence --action setPageBody --id "123456" --title "New Title" \
  --body "<p>New content</p>"
```

#### Attachments

```bash
# Upload attachment
atlas --type confluence --action addAttach --id "123456" --file "/path/to/file.pdf"

# Download all attachments from a page
atlas --type confluence --action downloadAttachments --id "123456"
```

#### Delete

```bash
# Archive (safer - can be restored)
atlas --type confluence --action archivePage --id "123456"

# Permanent deletion
atlas --type confluence --action deletePage --id "123456"
```

### Jira

```bash
atlas --type jira --action getIssue --key "PROJ-123"
atlas --type jira --action createIssue --summary "Title" --description "Details" --project "PROJ"
atlas --type jira --action getProject --key "PROJ"
```

## Output Format

All commands output a status prefix on the first line:

```
SUCCESS: Created page
  ID: 123456789
  Title: New Page
  Version: 1

FAILED: Page not found (HTTP 404)

WARNING: No matches found for "old text"
  ID: 123456789
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md).
