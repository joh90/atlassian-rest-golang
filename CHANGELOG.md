# Changelog

All notable changes to this project will be documented in this file.

## [0.0.10] - 2026-01-22

### Added
- Labels display in `getPage` output via metadata.labels expand
- Structured output for `getSpace` (Key, Name, Type, Status, Homepage)
- Space key in `listPages` results via expand=space
- Numbered list output for `downloadAttachments`

### Changed
- All commands now output SUCCESS/FAILED/WARNING prefix on first line
- Structured output with labeled fields (ID, Title, Version, etc.)
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
