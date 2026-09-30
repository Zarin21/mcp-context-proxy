package bench

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MockGitHubServer returns realistic large GitHub API responses
type MockGitHubServer struct{}

func (s *MockGitHubServer) HandleToolCall(toolName string, args json.RawMessage) json.RawMessage {
	switch toolName {
	case "search_repositories":
		return json.RawMessage(`{"items": [` + generateRepos(30) + `]}`)
	case "list_issues":
		return json.RawMessage(`{"items": [` + generateIssues(50) + `]}`)
	case "get_pull_request":
		return json.RawMessage(`{"title": "Fix context length issue", "body": "This PR fixes the context length issue by introducing a proxy.", "diff": "@@ -1,3 +1,3 @@\n- old\n+ new", "commits": 5, "changed_files": 1}`)
	default:
		return json.RawMessage(`{}`)
	}
}

func generateRepos(n int) string {
	var repos []string
	for i := 0; i < n; i++ {
		idStr := strconv.Itoa(i)
		repo := `{"id": ` + idStr + `, "node_id": "MDEwOlJlcG9zaXRvcnk=", "name": "repo-` + idStr + `", "full_name": "owner/repo-` + idStr + `", "private": false, "owner": {"login": "owner", "id": 1}, "html_url": "https://github.com/owner/repo", "description": "This is a realistic long description of the repository that takes up space.", "fork": false, "url": "https://api.github.com/repos/owner/repo", "forks_url": "https://api.github.com/repos/owner/repo/forks", "keys_url": "https://api.github.com/repos/owner/repo/keys{/key_id}", "collaborators_url": "https://api.github.com/repos/owner/repo/collaborators{/collaborator}", "teams_url": "https://api.github.com/repos/owner/repo/teams", "hooks_url": "https://api.github.com/repos/owner/repo/hooks", "issue_events_url": "https://api.github.com/repos/owner/repo/issues/events{/number}", "events_url": "https://api.github.com/repos/owner/repo/events", "assignees_url": "https://api.github.com/repos/owner/repo/assignees{/user}", "branches_url": "https://api.github.com/repos/owner/repo/branches{/branch}", "tags_url": "https://api.github.com/repos/owner/repo/tags", "blobs_url": "https://api.github.com/repos/owner/repo/git/blobs{/sha}", "git_tags_url": "https://api.github.com/repos/owner/repo/git/tags{/sha}", "git_refs_url": "https://api.github.com/repos/owner/repo/git/refs{/sha}", "trees_url": "https://api.github.com/repos/owner/repo/git/trees{/sha}", "statuses_url": "https://api.github.com/repos/owner/repo/statuses/{sha}", "languages_url": "https://api.github.com/repos/owner/repo/languages", "stargazers_url": "https://api.github.com/repos/owner/repo/stargazers", "contributors_url": "https://api.github.com/repos/owner/repo/contributors", "subscribers_url": "https://api.github.com/repos/owner/repo/subscribers", "subscription_url": "https://api.github.com/repos/owner/repo/subscription", "commits_url": "https://api.github.com/repos/owner/repo/commits{/sha}", "git_commits_url": "https://api.github.com/repos/owner/repo/git/commits{/sha}", "comments_url": "https://api.github.com/repos/owner/repo/comments{/number}", "issue_comment_url": "https://api.github.com/repos/owner/repo/issues/comments{/number}", "contents_url": "https://api.github.com/repos/owner/repo/contents/{+path}", "compare_url": "https://api.github.com/repos/owner/repo/compare/{base}...{head}", "merges_url": "https://api.github.com/repos/owner/repo/merges", "archive_url": "https://api.github.com/repos/owner/repo/{archive_format}{/ref}", "downloads_url": "https://api.github.com/repos/owner/repo/downloads", "issues_url": "https://api.github.com/repos/owner/repo/issues{/number}", "pulls_url": "https://api.github.com/repos/owner/repo/pulls{/number}", "milestones_url": "https://api.github.com/repos/owner/repo/milestones{/number}", "notifications_url": "https://api.github.com/repos/owner/repo/notifications{?since,all,participating}", "labels_url": "https://api.github.com/repos/owner/repo/labels{/name}", "releases_url": "https://api.github.com/repos/owner/repo/releases{/id}", "deployments_url": "https://api.github.com/repos/owner/repo/deployments", "created_at": "2021-01-01T00:00:00Z", "updated_at": "2021-01-01T00:00:00Z", "pushed_at": "2021-01-01T00:00:00Z", "git_url": "git://github.com/owner/repo.git", "ssh_url": "git@github.com:owner/repo.git", "clone_url": "https://github.com/owner/repo.git", "svn_url": "https://github.com/owner/repo", "homepage": "https://github.com", "size": 100, "stargazers_count": 100, "watchers_count": 100, "language": "Go", "has_issues": true, "has_projects": true, "has_downloads": true, "has_wiki": true, "has_pages": false, "forks_count": 100, "mirror_url": null, "archived": false, "disabled": false, "open_issues_count": 100, "license": {"key": "mit", "name": "MIT License", "spdx_id": "MIT", "url": "https://api.github.com/licenses/mit", "node_id": "MDc6TGljZW5zZTEz"}, "allow_forking": true, "is_template": false, "topics": ["go", "mcp", "context", "proxy"], "visibility": "public", "forks": 100, "open_issues": 100, "watchers": 100, "default_branch": "main", "permissions": {"admin": false, "maintain": false, "push": false, "triage": false, "pull": true}}`
		repos = append(repos, repo)
	}
	return strings.Join(repos, ",")
}

func generateIssues(n int) string {
	var issues []string
	for i := 0; i < n; i++ {
		idStr := strconv.Itoa(i)
		issue := `{"url": "https://api.github.com/repos/owner/repo/issues/1", "repository_url": "https://api.github.com/repos/owner/repo", "labels_url": "https://api.github.com/repos/owner/repo/issues/1/labels{/name}", "comments_url": "https://api.github.com/repos/owner/repo/issues/1/comments", "events_url": "https://api.github.com/repos/owner/repo/issues/1/events", "html_url": "https://github.com/owner/repo/issues/` + idStr + `", "id": ` + idStr + `, "node_id": "MDU6SXNzdWUx", "number": ` + idStr + `, "title": "Found a bug", "user": {"login": "user", "id": 1}, "labels": [{"id": 1, "node_id": "MDU6TGFiZWwx", "url": "https://api.github.com/repos/owner/repo/labels/bug", "name": "bug", "color": "d73a4a", "default": true, "description": "Something isn't working"}], "state": "open", "locked": false, "assignee": null, "assignees": [], "milestone": null, "comments": 0, "created_at": "2021-01-01T00:00:00Z", "updated_at": "2021-01-01T00:00:00Z", "closed_at": null, "author_association": "NONE", "active_lock_reason": null, "body": "I'm having a problem with this.", "reactions": {"url": "https://api.github.com/repos/owner/repo/issues/1/reactions", "total_count": 0, "+1": 0, "-1": 0, "laugh": 0, "hooray": 0, "confused": 0, "heart": 0, "rocket": 0, "eyes": 0}, "timeline_url": "https://api.github.com/repos/owner/repo/issues/1/timeline", "performed_via_github_app": null, "state_reason": null}`
		issues = append(issues, issue)
	}
	return strings.Join(issues, ",")
}

// MockFilesystemServer returns realistic file contents
type MockFilesystemServer struct{}

func (s *MockFilesystemServer) HandleToolCall(toolName string, args json.RawMessage) json.RawMessage {
	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, fmt.Sprintf("func ExampleFunction%d() {\n\tfmt.Println(\"This is an example function\")\n}\n", i))
	}
	content := strings.Join(lines, "\n")
	b, _ := json.Marshal(map[string]string{"content": content})
	return b
}

// MockLogServer returns realistic log output (5000+ lines)
type MockLogServer struct{}

func (s *MockLogServer) HandleToolCall(toolName string, args json.RawMessage) json.RawMessage {
	var logLines []string
	for i := 0; i < 5000; i++ {
		ts := fmt.Sprintf("2021-01-01T%02d:%02d:%02dZ", (i/3600)%24, (i/60)%60, i%60)
		switch {
		case i == 500:
			logLines = append(logLines, ts+" ERROR Database connection lost")
		case i == 2500:
			logLines = append(logLines, ts+" WARN Connection pool exhausted, retrying")
		case i == 3000:
			logLines = append(logLines, ts+" ERROR Timeout waiting for response from upstream service")
		case i%100 == 0:
			logLines = append(logLines, ts+" INFO Starting scheduled cleanup job id="+strconv.Itoa(i))
		case i%7 == 0:
			logLines = append(logLines, ts+" DEBUG Cache miss key=user:"+strconv.Itoa(i%1000))
		default:
			logLines = append(logLines, ts+" INFO Request received method=GET path=/health status=200 duration="+strconv.Itoa(i%50)+"ms")
		}
	}

	content := strings.Join(logLines, "\n")
	b, _ := json.Marshal(map[string]string{"logs": content})
	return b
}
