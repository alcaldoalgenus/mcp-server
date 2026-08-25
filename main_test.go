package main

import (
	"strings"
	"testing"
)

func TestBuildSearchQuery_EmptyBaseQuery(t *testing.T) {
	params := SearchParams{
		Owner: "owner",
		Repo:  "repo",
		State: "open",
		Type:  "issue",
	}
	got := BuildSearchQuery("", params)
	expected := "repo:owner/repo is:issue is:open"
	if got != expected {
		t.Errorf("expected '%s', got '%s'", expected, got)
	}
}

func TestBuildSearchQuery_PrecedenceAndNoDuplicates(t *testing.T) {
	params := SearchParams{
		Owner: "owner",
		Repo:  "repo",
		State: "open",
	}
	query := "repo:different/repo is:closed bug"
	got := BuildSearchQuery(query, params)
	// Should not append repo:owner/repo or is:open
	if strings.Contains(got, "repo:owner/repo") {
		t.Errorf("should not have appended repo:owner/repo, got '%s'", got)
	}
	if strings.Contains(got, "is:open") {
		t.Errorf("should not have appended is:open when is:closed is present, got '%s'", got)
	}
	if got != "repo:different/repo is:closed bug" {
		t.Errorf("unexpected result: '%s'", got)
	}
}

func TestBuildSearchQuery_QuotedAndNegatedFilters(t *testing.T) {
	params := SearchParams{
		Owner:  "owner",
		Repo:   "repo",
		Labels: []string{"bug", "good first issue"},
	}
	query := `"error loading" -repo:test/temp label:"good first issue"`
	got := BuildSearchQuery(query, params)
	if !strings.Contains(got, `"error loading"`) {
		t.Errorf("quoted search term lost: '%s'", got)
	}
	if !strings.Contains(got, `-repo:test/temp`) {
		t.Errorf("negated repo lost: '%s'", got)
	}
	if strings.Contains(got, `repo:owner/repo`) {
		t.Errorf("should not append repo when negated repo is present: '%s'", got)
	}
	if strings.Count(got, `"good first issue"`) != 1 {
		t.Errorf("duplicate label added: '%s'", got)
	}
	if !strings.Contains(got, `label:bug`) {
		t.Errorf("missing label:bug: '%s'", got)
	}
}

func TestBuildSearchQuery_WhitespaceSanitization(t *testing.T) {
	params := SearchParams{
		Owner: "owner",
		Repo:  "repo",
	}
	query := "   find   critical    bug   "
	got := BuildSearchQuery(query, params)
	expected := "find critical bug repo:owner/repo"
	if got != expected {
		t.Errorf("expected '%s', got '%s'", expected, got)
	}
}

func TestHandleAPIError(t *testing.T) {
	msg422 := HandleAPIError(422, "Validation Failed: Qualifier not recognized")
	if !strings.Contains(msg422, "HTTP 422") || !strings.Contains(msg422, "Validation Failed") {
		t.Errorf("unexpected 422 message: %s", msg422)
	}

	msg500 := HandleAPIError(500, "Internal Server Error")
	if !strings.Contains(msg500, "HTTP 500") {
		t.Errorf("unexpected 500 message: %s", msg500)
	}
}
