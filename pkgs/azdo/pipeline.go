package azdo

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"encoding/json"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var pipelineResults = []string{"succeeded", "failed", "partiallySucceeded", "canceled"}

type AzdoClient struct {
	authHeader        map[string][]string
	orgUrl            string
	defaultApiVersion string
}

type PipelineStateMsg pipelineState
type PipelinesFetchedMsg []list.Item
type PipelineIdMsg int

type pipelineState struct {
	IsRunning bool
	Stages    []StageState
}

type TaskState struct {
	Name   string
	Status string
	Log    string
	Id     string
}

type JobState struct {
	Name   string
	Status string
	Id     string
	Log    string
	Tasks  []TaskState
}

type StageState struct {
	Name   string
	Id     string
	Status string
	Log    string
	Jobs   []JobState
}

type Record struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parentId"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	Result    string    `json:"result"`
	Log       LogInfo   `json:"log"`
	StartTime time.Time `json:"startTime"`
	Children  []*Record
}

type LogInfo struct {
	Url string `json:"url"`
}

func NewAzdoClient(org, project, pat string) *AzdoClient {
	authHeader := map[string][]string{
		"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(":"+pat))},
		"Content-Type":  {"application/json"},
	}
	return &AzdoClient{
		authHeader:        authHeader,
		orgUrl:            fmt.Sprintf("https://dev.azure.com/%s/%s", org, project),
		defaultApiVersion: "api-version=7.2-preview.7",
	}
}

func (m *Model) getPipelineStatus(pipelineId int) (string, int) {
	apiURL := fmt.Sprintf("%s/_apis/build/builds?definitions=%d&queryOrder=queueTimeDescending&$top=1&%s", m.azdoClient.orgUrl, pipelineId, m.azdoClient.defaultApiVersion)
	req, err := http.NewRequest("GET", apiURL, nil)
	req.Header = m.azdoClient.authHeader
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var r map[string]interface{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		panic(err)
	}
	runCount := int(r["count"].(float64))
	if runCount == 0 {
		log2file("No runs found\n")
		return "noRuns", 0
	}
	run := r["value"].([]interface{})[0].(map[string]interface{})
	runStatus := run["status"].(string)
	if runStatus == "completed" {
		return run["result"].(string), int(run["id"].(float64))
	}
	return run["status"].(string), int(run["id"].(float64))
}

func (m *Model) RunOrFollowPipeline(id int, runNew bool) tea.Msg {
	apiURL := fmt.Sprintf("%s/_apis/pipelines/%d/runs?%s", m.azdoClient.orgUrl, id, "api-version=7.1-preview.1")
	if status, runId := m.getPipelineStatus(id); !slices.Contains(pipelineResults, status) && !runNew {
		return PipelineIdMsg(runId)
	}

	runParameters := map[string]interface{}{
		"resources": map[string]interface{}{
			"repositories": map[string]interface{}{
				"self": map[string]interface{}{
					"refName": "refs/heads/master",
				},
			},
		},
	}
	runParametersJson, _ := json.Marshal(runParameters)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(runParametersJson))
	if err != nil {
		panic(err)
	}

	// Add authorization header
	req.Header = m.azdoClient.authHeader

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var r map[string]interface{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		panic(err)
	}
	pipelineId := int(r["id"].(float64))
	return PipelineIdMsg(pipelineId)

}

func (c *AzdoClient) getPipelineState(runId int, wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(wait)
		apiURL := fmt.Sprintf("%s/_apis/build/builds/%d/timeline?%s", c.orgUrl, runId, "api-version=7.2-preview.2")
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			panic(err)
		}

		// Add authorization header
		req.Header = c.authHeader

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var r map[string]interface{}
		err = json.Unmarshal(body, &r)
		if err != nil {
			panic(err)
		}

		var records []Record
		// map r['records'] to records
		recordsByte, err := json.Marshal(r["records"])
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(recordsByte, &records); err != nil {
			panic(err)
		}
		ps := c.fillPipelineStatus(records)
		return PipelineStateMsg(ps)
	}
}

func (c *AzdoClient) fillPipelineStatus(records []Record) pipelineState {
	// sort records by startTime
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartTime.Before(records[j].StartTime)
	})
	// Map records by ID and initialize children slice
	recordMap := make(map[string]*Record)
	for i, _ := range records {
		recordMap[records[i].ID] = &records[i]
		records[i].Children = make([]*Record, 0)
	}

	// Build tree by appending each record to the children of its parent and also fill recordsState
	var recordsState []string
	for i := range records {
		recordsState = append(recordsState, records[i].State)
		if parent, ok := recordMap[records[i].ParentID]; ok {
			parent.Children = append(parent.Children, &records[i])
		}
	}
	var ps pipelineState
	ps.IsRunning = slices.Contains(recordsState, "inProgress") || slices.Contains(recordsState, "pending")
	for _, record := range records {
		if record.Type == "Stage" {
			status := getRecordStatus(record)
			stageState := StageState{
				Name:   record.Name,
				Id:     record.ID,
				Status: status,
			}
			for _, child := range record.Children {
				if child.Type == "Phase" && len(child.Children) > 0 {
					status := getRecordStatus(*child.Children[0])
					jobState := JobState{
						Name:   child.Children[0].Name,
						Status: status,
						Id:     child.Children[0].ID,
					}
					for _, task := range child.Children[0].Children {
						status := getRecordStatus(*task)
						taskState := TaskState{
							Name:   task.Name,
							Status: status,
							Id:     task.ID,
							Log:    c.getTaskLog(task),
						}
						jobState.Tasks = append(jobState.Tasks, taskState)
						jobState.Log += taskState.Log
					}
					stageState.Jobs = append(stageState.Jobs, jobState)
					stageState.Log += jobState.Log
				}
			}
			ps.Stages = append(ps.Stages, stageState)
		}
	}
	return ps
}

func (c *AzdoClient) getTaskLog(task *Record) string {
	if task.Log.Url == "" {
		return ""
	}
	req, err := http.NewRequest("GET", task.Log.Url, nil)
	if err != nil {
		panic(err)
	}

	// Add authorization header
	req.Header = c.authHeader
	// set header to get utf-8 response
	req.Header.Add("Accept-Charset", "utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	return processLog(resp.Body)
}

func processLog(text io.ReadCloser) string {
	scanner := bufio.NewScanner(text)
	var processedText string
	lineNum := 1
	maxDigits := len(fmt.Sprintf("%d", 10000))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		var newLine string
		if len(parts) > 1 {
			newLine = fmt.Sprintf("%*d | %s", maxDigits, lineNum, parts[1])
		} else {
			newLine = fmt.Sprintf("%*d | %s", maxDigits, lineNum, line)
		}
		processedText += newLine + "\n"
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return processedText
}

func (m *Model) FetchPipelines(wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(wait)
		apiURL := fmt.Sprintf("%s/_apis/pipelines?api-version=6.0-preview.1", m.azdoClient.orgUrl)
		client := &http.Client{}
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			panic(err)
		}
		req.Header = m.azdoClient.authHeader
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		pipelineList := []list.Item{}
		for _, pipeline := range result["value"].([]interface{}) {
			pipelineObj := pipeline.(map[string]interface{})
			pipelineName := pipelineObj["name"].(string)
			pipelineId := int(pipelineObj["id"].(float64))
			status, _ := m.getPipelineStatus(pipelineId)
			symbol := m.getSymbol(status)
			pipelineList = append(pipelineList, PipelineItem{Title: pipelineName, Desc: pipelineId, Status: status, Symbol: symbol})
		}
		return PipelinesFetchedMsg(pipelineList)
	}
}

func getRecordStatus(record Record) string {
	if record.Result != "" {
		return record.Result
	}
	return record.State
}
