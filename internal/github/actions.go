package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
)

const apiRoot = "https://api.github.com"

var tokenMu sync.Mutex
var cachedToken string

func tokenValue() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if cachedToken != "" {
		return cachedToken, nil
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token failed (did you run `gh auth login`?): %w", err)
	}
	cachedToken = strings.TrimSpace(string(out))
	if cachedToken == "" {
		return "", fmt.Errorf("empty gh auth token output")
	}
	return cachedToken, nil
}

type Branch struct {
	Name        string
	IsDefault   bool
	IsProtected bool
}

type branchWire struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

func ListBranches(owner, repo string) ([]Branch, error) {
	tok, err := tokenValue()
	if err != nil {
		return nil, err
	}

	var meta struct {
		Default string `json:"default_branch"`
	}
	if _, _, err := GETJSON(tok, fmt.Sprintf("/repos/%s/%s", owner, repo), &meta); err != nil {
		return nil, err
	}

	var aggregated []Branch
	nextURL := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=100", apiRoot, owner, repo)

	for nextURL != "" {
		req, err := http.NewRequest(http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}
		fillHeaders(req, tok)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("branches endpoint %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var page []branchWire
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}

		for _, bw := range page {
			aggregated = append(aggregated, Branch{
				Name:        bw.Name,
				IsDefault:   bw.Name == meta.Default,
				IsProtected: bw.Protected,
			})
		}

		nextURL = parseNext(resp.Header.Get("Link"))
	}

	return aggregated, nil
}

type WorkflowRun struct {
	ID         int64
	Display    string
	Workflow   string
	Status     string
	Conclusion *string
	Branch     string
	HeadSha    string
	CreatedAt  string
	UpdatedAt  string
	HTMLURL    string
	Event      string
	Actor      string
	RunNumber  int
}

func ListRuns(owner, repo, branch string, perPage int) ([]WorkflowRun, error) {
	tok, err := tokenValue()
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=%d", owner, repo, perPage)
	if branch != "" {
		q += "&branch=" + url.QueryEscape(branch)
	}

	var envelope struct {
		WorkflowRuns []struct {
			ID           int64   `json:"id"`
			Name         string  `json:"name"`
			DisplayTitle string  `json:"display_title"`
			Status       string  `json:"status"`
			Conclusion   *string `json:"conclusion"`
			HeadBranch   string  `json:"head_branch"`
			HeadSha      string  `json:"head_sha"`
			CreatedAt    string  `json:"created_at"`
			UpdatedAt    string  `json:"updated_at"`
			HTMLURL      string  `json:"html_url"`
			Event        string  `json:"event"`
			Actor        *struct {
				Login string `json:"login"`
			} `json:"actor"`
			RunNumber    int    `json:"run_number"`
			WorkflowName string `json:"workflow_name"`
		} `json:"workflow_runs"`
	}

	if _, _, err := GETJSON(tok, q, &envelope); err != nil {
		return nil, err
	}

	out := make([]WorkflowRun, 0, len(envelope.WorkflowRuns))
	for _, r := range envelope.WorkflowRuns {
		display := strings.TrimSpace(r.DisplayTitle)
		if display == "" {
			display = r.Name
		}
		wfTitle := strings.TrimSpace(r.WorkflowName)
		if wfTitle == "" {
			wfTitle = "workflow"
		}
		actor := "unknown"
		if r.Actor != nil {
			actor = r.Actor.Login
		}

		out = append(out, WorkflowRun{
			ID:         r.ID,
			Display:    display,
			Workflow:   wfTitle,
			Status:     r.Status,
			Conclusion: r.Conclusion,
			Branch:     r.HeadBranch,
			HeadSha:    r.HeadSha,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			HTMLURL:    r.HTMLURL,
			Event:      r.Event,
			Actor:      actor,
			RunNumber:  r.RunNumber,
		})
	}
	return out, nil
}

type Step struct {
	Name        string
	Status      string
	Conclusion  *string
	Number      int
	StartedAt   *string
	CompletedAt *string
}

type Job struct {
	ID          int64
	Name        string
	Status      string
	Conclusion  *string
	StartedAt   *string
	CompletedAt *string
	Steps       []Step
	RunnerName  *string
}

func ListJobs(owner string, repo string, runID int64) ([]Job, error) {
	tok, err := tokenValue()
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Jobs []struct {
			ID          int64   `json:"id"`
			Name        string  `json:"name"`
			Status      string  `json:"status"`
			Conclusion  *string `json:"conclusion"`
			StartedAt   *string `json:"started_at"`
			CompletedAt *string `json:"completed_at"`
			RunnerName  *string `json:"runner_name"`
			Steps       []struct {
				Name        string  `json:"name"`
				Status      string  `json:"status"`
				Conclusion  *string `json:"conclusion"`
				Number      int     `json:"number"`
				StartedAt   *string `json:"started_at"`
				CompletedAt *string `json:"completed_at"`
			} `json:"steps"`
		} `json:"jobs"`
	}

	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs?per_page=100", owner, repo, runID)
	if _, _, err := GETJSON(tok, path, &envelope); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(envelope.Jobs))
	for _, job := range envelope.Jobs {
		steps := make([]Step, 0, len(job.Steps))
		for _, s := range job.Steps {
			steps = append(steps, Step{
				Name:        s.Name,
				Status:      s.Status,
				Conclusion:  s.Conclusion,
				Number:      s.Number,
				StartedAt:   s.StartedAt,
				CompletedAt: s.CompletedAt,
			})
		}
		jobs = append(jobs, Job{
			ID:          job.ID,
			Name:        job.Name,
			Status:      job.Status,
			Conclusion:  job.Conclusion,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
			Steps:       steps,
			RunnerName:  job.RunnerName,
		})
	}
	return jobs, nil
}

func DownloadJobLogs(owner, repo string, jobID int64) ([]byte, error) {
	tok, err := tokenValue()
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/repos/%s/%s/actions/jobs/%d/logs", apiRoot, owner, repo, jobID)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	fillHeaders(req, tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusMultipleChoices {
		return data, fmt.Errorf("logs endpoint %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, readErr
}

type Workflow struct {
	ID    int64
	Name  string
	Path  string
	State string
}

func ListWorkflows(owner, repo string) ([]Workflow, error) {
	tok, err := tokenValue()
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Workflows []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Path  string `json:"path"`
			State string `json:"state"`
		} `json:"workflows"`
	}

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows?per_page=100", owner, repo)
	if _, _, err := GETJSON(tok, path, &envelope); err != nil {
		return nil, err
	}

	out := make([]Workflow, 0, len(envelope.Workflows))
	for _, w := range envelope.Workflows {
		if strings.EqualFold(w.State, "active") {
			out = append(out, Workflow{ID: w.ID, Name: w.Name, Path: w.Path, State: w.State})
		}
	}
	return out, nil
}

func DispatchWorkflow(owner, repo string, workflowID int64, ref string, inputs map[string]string) error {
	tok, err := tokenValue()
	if err != nil {
		return err
	}

	body := map[string]any{"ref": ref}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%d/dispatches", owner, repo, workflowID)
	_, err = POSTBODY(tok, path, &buf)
	return err
}

func CancelRun(owner, repo string, runID int64) error {
	tok, err := tokenValue()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", owner, repo, runID)
	_, err = POSTEMPTY(tok, path)
	return err
}

func Rerun(owner, repo string, runID int64) error {
	tok, err := tokenValue()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun", owner, repo, runID)
	_, err = POSTEMPTY(tok, path)
	return err
}

func RerunFailed(owner, repo string, runID int64) error {
	tok, err := tokenValue()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun-failed-jobs", owner, repo, runID)
	_, err = POSTEMPTY(tok, path)
	return err
}

// CollectRunLogs merges per-job plaintext logs similarly to lazyrun’s JS implementation.
func CollectRunLogs(owner, repo string, runID int64) (string, error) {
	jobs, err := ListJobs(owner, repo, runID)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, job := range jobs {
		sb.WriteByte('\n')
		fmt.Fprintf(&sb, "━━━ Job: %s (%s", job.Name, job.Status)
		if job.Conclusion != nil && *job.Conclusion != "" {
			fmt.Fprintf(&sb, " / %s", *job.Conclusion)
		}
		sb.WriteString(") ━━━\n")

		payload, err := DownloadJobLogs(owner, repo, job.ID)
		if err != nil {
			fmt.Fprintf(&sb, "(failed to fetch logs: %v)\n", err)
			continue
		}

		body := strings.TrimSpace(string(payload))
		if body != "" {
			sb.WriteString(body)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}

	return sb.String(), nil
}

func parseNext(linkHeader string) string {
	for _, chunk := range strings.Split(linkHeader, ",") {
		parts := strings.Split(chunk, ";")
		if len(parts) != 2 {
			continue
		}
		urlToken := strings.Trim(strings.TrimSpace(parts[0]), "<>")
		if strings.Contains(parts[1], `rel="next"`) || strings.Contains(parts[1], `rel='next'`) {
			return urlToken
		}
	}
	return ""
}

func fillHeaders(req *http.Request, tok string) {
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lazyrun")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func RAWGET(tok, path string) ([]byte, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, apiRoot+path, nil)
	if err != nil {
		return nil, nil, err
	}
	fillHeaders(req, tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusMultipleChoices {
		return data, resp.Header, fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, resp.Header, err
}

func GETJSON(tok string, path string, out any) ([]byte, http.Header, error) {
	body, hdr, err := RAWGET(tok, path)
	if err != nil {
		return body, hdr, err
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return body, hdr, nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return body, hdr, err
	}
	return body, hdr, nil
}

func POSTEMPTY(tok, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, apiRoot+path, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	fillHeaders(req, tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusMultipleChoices {
		return data, fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func POSTBODY(tok, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, apiRoot+path, body)
	if err != nil {
		return nil, err
	}
	fillHeaders(req, tok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusMultipleChoices {
		return data, fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}
