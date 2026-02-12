# Changelog

All notable changes to this project will be documented in this file.

## [0.0.11] - 2026-02-12

### Added
- `searchUsers`: Search for users with CQL (by Jiachao)
  - Single-line pipe-separated output: Account ID | Name | Email | Type
  - AccountType field to distinguish users from apps

### Changed
- `EMail` → `Email` in User model (JSON tag unchanged, no API impact)
- Error handling: CLI-exposed service functions now return
  (result, httpStatus, errorMessage) instead of panicking or failing silently
  - SearchCQL (search, listPages)
  - GetSpacePages (listPages)
  - GetPageTitleKey (getPage by title)
  - GetSpace (getSpace, @home resolution)
  - AddLabels (addLabel, createPage --labels)
  - AddAttachment (addAttach)
  - DownloadAttachments (downloadAttachments)
- `@home` resolution prints clear error instead of passing empty
  parent ID (was: cryptic "ContentId string must not be null")
- `downloadAttachments` with bad page ID now returns FAILED
  instead of misleading "SUCCESS: Downloaded 0"

### Fixed
- `getPage` by title: URL-encode title parameter (was: titles with spaces
  caused HTTP request to hang)
- `getPage` by title: parse API response as ContentArray instead of single
  Content (was: always returned "Page not found")

## [0.0.10] - 2026-01-22

### Added
- Labels display in `getPage` output via metadata.labels expand
- Structured output for `getSpace` (Key, Name, Type, Status, Homepage)
- Space key in `listPages` results via expand=space
- Numbered list output for `downloadAttachments`

### Changed
- All commands now output SUCCESS/FAILED/WARNING prefix on first line
- Structured output with labeled fields (ID, Title, Version, etc.)
- `updatePage` now shows match count ("Replaced N occurrence(s)") or WARNING if no matches found
- `GetPage` and `CreateContent` APIs return (content, httpStatus, errorMessage)

### Fixed
- `downloadAttachments` returning empty titles
- Path traversal vulnerability in `downloadAttachments` (sanitize filename)
- Debug JSON dump removed from `getSpace`

## [0.0.9.2] - 2026-01-12

### Fixed
- Attachment upload (`addAttach` action)
  - Fix form field name: "multipart/form-data" → "file"
  - Use filepath.Base() for filename instead of full path
  - Close multipart writer before sending request
  - Parse API response correctly (returns {"results": [...]} array)
  - Add proper error handling throughout

## [0.0.9.1] - 2025-12-11

### Added
- `search`: CQL search with URL encoding
- `listPages`: List all pages in a space
- `updatePage`: Find/replace text in a page
- `setPageBody`: Replace entire page body
- `archivePage`: Archive a page (safer than delete)
- `deletePage`: Permanently delete a page
- New CLI arguments: `--find`, `--replace`, `--cql`, `--limit`

### Fixed
- EditPage serialization error (avoids "ContentId string must not be null" error)
- DeletePage to return proper success/failure with HTTP status check
