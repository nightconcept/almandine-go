package source

import (
	"fmt"
	"net/url"
	"strings"
)

// ParsedSourceInfo holds the details extracted from a source URL.
type ParsedSourceInfo struct {
	RawURL            string // The raw URL to download the file content
	CanonicalURL      string // The canonical representation (e.g., github:owner/repo/path/to/file@ref)
	Ref               string // The commit hash, branch, or tag
	Provider          string // e.g., "github"
	Owner             string
	Repo              string
	PathInRepo        string
	SuggestedFilename string
}

// ParseSourceURL analyzes the input source URL string and returns structured information.
// It currently prioritizes GitHub URLs.
func ParseSourceURL(sourceURL string) (*ParsedSourceInfo, error) {
	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source URL '%s': %w", sourceURL, err)
	}

	if strings.ToLower(u.Hostname()) == "github.com" {
		return parseGitHubURL(u)
	}

	// Placeholder for other providers or generic git repositories
	return nil, fmt.Errorf("unsupported source URL host: %s. Only GitHub URLs are currently supported", u.Hostname())
}

// parseGitHubURL handles the specifics of parsing GitHub URLs.
func parseGitHubURL(u *url.URL) (*ParsedSourceInfo, error) {
	// Path components: /<owner>/<repo>/<type>/<ref>/<path_to_file>
	// or /<owner>/<repo>/raw/<ref>/<path_to_file>
	// or /<owner>/<repo> (if wanting to point to a whole repo, though we expect a file)
	// Example: https://github.com/owner/repo/blob/main/path/to/file.go
	// Example: https://github.com/owner/repo/raw/develop/script.sh
	// Example: https://raw.githubusercontent.com/owner/repo/main/path/to/file.go

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if strings.ToLower(u.Hostname()) == "raw.githubusercontent.com" {
		// Format: /<owner>/<repo>/<ref>/<path_to_file>
		if len(pathParts) < 4 {
			return nil, fmt.Errorf("invalid GitHub raw content URL path: %s. Expected format: /<owner>/<repo>/<ref>/<path_to_file>", u.Path)
		}
		owner := pathParts[0]
		repo := pathParts[1]
		ref := pathParts[2]
		filePathInRepo := strings.Join(pathParts[3:], "/")
		filename := pathParts[len(pathParts)-1]

		canonicalURL := fmt.Sprintf("github:%s/%s/%s@%s", owner, repo, filePathInRepo, ref)
		// The input URL is already the raw download URL
		return &ParsedSourceInfo{
			RawURL:            u.String(),
			CanonicalURL:      canonicalURL,
			Ref:               ref,
			Provider:          "github",
			Owner:             owner,
			Repo:              repo,
			PathInRepo:        filePathInRepo,
			SuggestedFilename: filename,
		}, nil
	}

	// Regular github.com URL
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL path: %s. Expected at least /<owner>/<repo>", u.Path)
	}

	owner := pathParts[0]
	repo := pathParts[1]
	var ref, filePathInRepo, rawURL, filename string

	// /<owner>/<repo> - this case is not directly supported as we need a file.
	// We could default to fetching default branch's project file or error out.
	// For now, let's assume the URL is more specific.

	// Check for patterns like /blob/, /tree/, /raw/
	// /<owner>/<repo>/blob/<ref>/<path_to_file>
	// /<owner>/<repo>/raw/<ref>/<path_to_file> (less common for user input but possible)
	if len(pathParts) >= 4 && (pathParts[2] == "blob" || pathParts[2] == "tree" || pathParts[2] == "raw") {
		if len(pathParts) < 5 {
			return nil, fmt.Errorf("incomplete GitHub URL path: %s. Expected /<owner>/<repo>/<type>/<ref>/<path_to_file>", u.Path)
		}
		refType := pathParts[2] // blob, tree, or raw
		ref = pathParts[3]
		filePathInRepo = strings.Join(pathParts[4:], "/")
		filename = pathParts[len(pathParts)-1]

		if refType == "tree" {
			return nil, fmt.Errorf("direct links to GitHub trees are not supported for adding single files: %s", u.String())
		}
		// Normalize to raw content URL
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, filePathInRepo)

	} else {
		// Try to extract ref and path if a shorthand URL is given, e.g., github.com/owner/repo/file.txt@main
		// This means the path might contain '@' for the ref.
		// If no type (blob/tree/raw) and path has more than 2 parts (/owner/repo/...)
		// we assume the structure is /owner/repo/path@ref or /owner/repo/path (implies default branch)
		// For Almandine, we will require the ref to be explicit in the path via @ or a more specific URL type.

		// The PRD implies a specific ref might be part of the source_url string itself,
		// rather than always inferred from structure.
		// "Extract commit hash/ref if present."

		// Let's use a regex to find ref in the path for URLs like:
		// github.com/user/repo/path/to/file.txt@v1.0.0
		// github.com/user/repo/path/to/file.txt@commitsha
		// github.com/user/repo/path/to/file.txt (implies default branch, but we need a way to get it)
		// For now, we'll assume if no /blob/ or /raw/ and no @ in path, we might need to query default branch.
		// Task 2.2 focuses on parsing, let's assume ref is extractable or default is handled later.

		// Simplified: if not blob/raw/tree, the path from part 2 onwards is file path, ref needs to be found or assumed.
		// For now, let's require a ref to be specified using '@' if not using blob/raw URLs.
		// Example: github.com/owner/repo/some/file.go@main

		potentialPathWithRef := strings.Join(pathParts[2:], "/")
		atSymbolIndex := strings.LastIndex(potentialPathWithRef, "@")

		if atSymbolIndex != -1 {
			filePathInRepo = potentialPathWithRef[:atSymbolIndex]
			ref = potentialPathWithRef[atSymbolIndex+1:]
			pathElements := strings.Split(filePathInRepo, "/")
			if len(pathElements) > 0 {
				filename = pathElements[len(pathElements)-1]
			} else {
				filename = "default_filename" // Or error if path is empty
			}
		} else {
			// No explicit ref in path, no blob/raw.
			// This case is ambiguous without fetching default branch info from GitHub API.
			// For now, we'll error or require a more specific URL.
			// Let's assume default ref = "main" for now, can be enhanced.
			// This part might need adjustment based on how we fetch default branch or if we enforce explicit refs always.
			// For simplicity in this step, let's error if ref is not obvious.
			return nil, fmt.Errorf("ambiguous GitHub URL: %s. Specify a branch/tag/commit via '@' (e.g., file.txt@main) or use a full /blob/ or /raw/ URL", u.String())
		}
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, filePathInRepo)
	}

	if filePathInRepo == "" {
		return nil, fmt.Errorf("file path in repository could not be determined from URL: %s", u.String())
	}
	if ref == "" {
		// This should ideally fetch the default branch, but that's an external call.
		// For parsing, we'll state that ref couldn't be determined if not explicitly part of URL.
		return nil, fmt.Errorf("ref (branch, tag, commit) could not be determined from URL: %s. Please specify it", u.String())
	}

	canonicalURL := fmt.Sprintf("github:%s/%s/%s@%s", owner, repo, filePathInRepo, ref)

	return &ParsedSourceInfo{
		RawURL:            rawURL,
		CanonicalURL:      canonicalURL,
		Ref:               ref,
		Provider:          "github",
		Owner:             owner,
		Repo:              repo,
		PathInRepo:        filePathInRepo,
		SuggestedFilename: filename,
	}, nil
}
