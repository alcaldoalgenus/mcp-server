package main

import (
	"fmt"
	"regexp"
	"strings"
)

// SearchParams represents the parameters passed to search_issues tool.
type SearchParams struct {
	Query    string   `json:"query,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	Repo     string   `json:"repo,omitempty"`
	State    string   `json:"state,omitempty"`
	Type     string   `json:"type,omitempty"`
	Author   string   `json:"author,omitempty"`
	Assignee string   `json:"assignee,omitempty"`
	Org      string   `json:"org,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// Qualifier represents an extracted search qualifier.
type Qualifier struct {
	Key   string
	Value string
	Raw   string
}

var qualifierRegex = regexp.MustCompile(`(?i)(?:^|\s)(-?[\w-]+):(?:"([^"]+)"|([^\s]+))`)
var multiSpaceRegex = regexp.MustCompile(`\s+`)

// ParseQualifiers extracts qualifier key-value pairs from a raw query string.
func ParseQualifiers(query string) []Qualifier {
	matches := qualifierRegex.FindAllStringSubmatch(query, -1)
	var qualifiers []Qualifier
	for _, match := range matches {
		if len(match) >= 4 {
			key := strings.ToLower(match[1])
			val := match[2]
			if val == "" {
				val = match[3]
			}
			qualifiers = append(qualifiers, Qualifier{
				Key:   key,
				Value: val,
				Raw:   strings.TrimSpace(match[0]),
			})
		}
	}
	return qualifiers
}

// formatQualifierValue wraps a qualifier value in quotes if it contains spaces.
func formatQualifierValue(val string) string {
	if strings.Contains(val, " ") && !strings.HasPrefix(val, "\"") && !strings.HasSuffix(val, "\"") {
		return fmt.Sprintf("\"%s\"", val)
	}
	return val
}

// BuildSearchQuery builds a clean, harmonized query string without redundant or conflicting qualifiers.
func BuildSearchQuery(rawQuery string, params SearchParams) string {
	trimmedQuery := strings.TrimSpace(rawQuery)
	qualifiers := ParseQualifiers(trimmedQuery)

	qualifierKeys := make(map[string]bool)
	qualifierValues := make(map[string][]string)
	for _, q := range qualifiers {
		qualifierKeys[q.Key] = true
		qualifierValues[q.Key] = append(qualifierValues[q.Key], strings.ToLower(q.Value))
	}

	var extraTerms []string

	// Handle repo & org
	hasRepoQualifier := qualifierKeys["repo"] || qualifierKeys["-repo"]
	hasOrgQualifier := qualifierKeys["org"] || qualifierKeys["-org"]

	if !hasRepoQualifier && params.Owner != "" && params.Repo != "" {
		extraTerms = append(extraTerms, fmt.Sprintf("repo:%s/%s", params.Owner, params.Repo))
	} else if !hasRepoQualifier && !hasOrgQualifier && params.Org != "" {
		extraTerms = append(extraTerms, fmt.Sprintf("org:%s", params.Org))
	}

	// Handle type (is:issue / is:pr / type:issue / type:pr)
	hasType := qualifierKeys["type"] || qualifierKeys["-type"]
	hasIsIssue := false
	for _, v := range qualifierValues["is"] {
		if v == "issue" || v == "pr" || v == "pull-request" {
			hasIsIssue = true
			break
		}
	}
	if !hasType && !hasIsIssue && params.Type != "" {
		typeVal := strings.ToLower(params.Type)
		if typeVal == "pr" || typeVal == "pull-request" {
			extraTerms = append(extraTerms, "is:pr")
		} else if typeVal == "issue" {
			extraTerms = append(extraTerms, "is:issue")
		} else {
			extraTerms = append(extraTerms, fmt.Sprintf("type:%s", formatQualifierValue(params.Type)))
		}
	}

	// Handle state (is:open / is:closed / state:open / state:closed)
	hasStateQualifier := qualifierKeys["state"] || qualifierKeys["-state"]
	for _, v := range qualifierValues["is"] {
		if v == "open" || v == "closed" || v == "merged" || v == "unmerged" || v == "draft" {
			hasStateQualifier = true
			break
		}
	}
	if !hasStateQualifier && params.State != "" {
		stateLower := strings.ToLower(params.State)
		if stateLower == "open" {
			extraTerms = append(extraTerms, "is:open")
		} else if stateLower == "closed" {
			extraTerms = append(extraTerms, "is:closed")
		} else if stateLower != "all" {
			extraTerms = append(extraTerms, fmt.Sprintf("state:%s", formatQualifierValue(params.State)))
		}
	}

	// Handle author
	if !qualifierKeys["author"] && !qualifierKeys["-author"] && params.Author != "" {
		extraTerms = append(extraTerms, fmt.Sprintf("author:%s", formatQualifierValue(params.Author)))
	}

	// Handle assignee
	if !qualifierKeys["assignee"] && !qualifierKeys["-assignee"] && params.Assignee != "" {
		extraTerms = append(extraTerms, fmt.Sprintf("assignee:%s", formatQualifierValue(params.Assignee)))
	}

	// Handle labels
	existingLabels := make(map[string]bool)
	for _, val := range qualifierValues["label"] {
		existingLabels[val] = true
	}
	for _, label := range params.Labels {
		if label != "" && !existingLabels[strings.ToLower(label)] {
			extraTerms = append(extraTerms, fmt.Sprintf("label:%s", formatQualifierValue(label)))
		}
	}

	// Join and sanitize
	var finalTerms []string
	if trimmedQuery != "" {
		finalTerms = append(finalTerms, trimmedQuery)
	}
	finalTerms = append(finalTerms, extraTerms...)

	fullQuery := strings.Join(finalTerms, " ")
	fullQuery = multiSpaceRegex.ReplaceAllString(fullQuery, " ")
	return strings.TrimSpace(fullQuery)
}

// HandleAPIError formats upstream HTTP validation errors into clear client error messages.
func HandleAPIError(statusCode int, upstreamMessage string) string {
	if statusCode == 422 {
		return fmt.Sprintf("Search query validation failed (HTTP 422): %s. Please check search qualifiers and query syntax.", upstreamMessage)
	}
	return fmt.Sprintf("Search request failed with status HTTP %d: %s", statusCode, upstreamMessage)
}

func main() {
	fmt.Println("MCP Server Query Builder initialized.")
}
