package azdo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	tea "github.com/charmbracelet/bubbletea"
)

type PRMsg string

func (m *Model) OpenPR(from, to, title, description string) tea.Msg {
	apiUrl := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests?api-version=7.1", m.azdoClient.orgUrl, m.repositoryId)
	log2file(fmt.Sprintf("openPR apiUrl: %s\n", apiUrl))
	prParams := map[string]interface{}{
		"sourceRefName": fmt.Sprintf("%s", from),
		"targetRefName": fmt.Sprintf("%s", to),
		"title":         title,
		"description":   description,
	}
	prParamsJson, _ := json.Marshal(prParams)
	log2file(fmt.Sprintf("openPR prParamsJson: %s\n", string(prParamsJson)))
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(prParamsJson))
	if err != nil {
		panic(err)
	}
	req.Header = m.azdoClient.authHeader
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// get body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log2file(fmt.Sprintf("openPR response: %s\n", string(body)))
	// check for 409 conflict response
	if resp.StatusCode == 409 {
		return PRMsg("PR already exists.")
	}
	return PRMsg("Successfully opened PR.")
}
