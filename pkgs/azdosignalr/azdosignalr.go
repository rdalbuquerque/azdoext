package azdosignalr

import (
	"azdoext/pkgs/logger"
	"azdoext/pkgs/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a SignalR message
type Message struct {
	H string   `json:"H"`
	M string   `json:"M"`
	A []Detail `json:"A"`
}

// Detail represents the details of a SignalR message
type Detail struct {
	Lines            []string `json:"lines"`
	TimelineID       string   `json:"timelineId"`
	TimelineRecordID string   `json:"timelineRecordId"`
	StepRecordID     string   `json:"stepRecordId"`
	BuildID          int      `json:"buildId"`
	Build            Build    `json:"build"`
}

type Build struct {
	Status string `json:"status"`
	Result string `json:"result"`
}

// SignalRResponse represents a SignalR response
type SignalRResponse struct {
	C string    `json:"C"`
	M []Message `json:"M"`
}

type SignalRClient struct {
	Conn         *websocket.Conn
	logger       *logger.Logger
	Organization string
	AccountID    string
	ProjectID    string
}

type negotiateResponse struct {
	ConnectionToken string `json:"ConnectionToken"`
}

func GetConnectionParameters(organization string, accountID string, projectID string) (string, http.Header, error) {
	authHeader := fetchAuthHeader()
	connectionToken, err := fetchConnectionToken(authHeader, organization, projectID)
	if err != nil {
		return "", nil, err
	}

	contextToken := accountID
	queryParams := url.Values{}
	queryParams.Add("transport", "webSockets")
	queryParams.Add("contextToken", contextToken)
	queryParams.Add("connectionToken", connectionToken)

	signalrURL := url.URL{
		Scheme:   "wss",
		Host:     "dev.azure.com",
		Path:     fmt.Sprintf("_signalr/%s/_apis/%s/signalr/connect", organization, projectID),
		RawQuery: queryParams.Encode(),
	}

	header := http.Header{}
	header.Add("Authorization", authHeader)

	return signalrURL.String(), header, nil
}

func sendHandshake(c *websocket.Conn) error {
	handshake := map[string]interface{}{
		"protocol": "json",
		"version":  1,
	}
	err := c.WriteJSON(handshake)
	if err != nil {
		return err
	}
	return nil
}

// NewSignalRConn initializes and returns a new websocket connection with Azure Devops SignalR endpoint
func NewSignalR(organization, accountID, projectID string) *SignalRClient {
	logger := logger.NewLogger("azdosignalr.log")

	return &SignalRClient{
		Organization: organization,
		AccountID:    accountID,
		ProjectID:    projectID,
		logger:       logger,
	}
}

// fetchAuthHeader fetches the authorization header
func fetchAuthHeader() string {
	pat := os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")
	basicAuth := base64.StdEncoding.EncodeToString([]byte(":" + pat))
	return "Basic " + basicAuth
}

// fetchConnectionToken fetches the connection token
func fetchConnectionToken(authHeader, organization, projectID string) (string, error) {
	queryParams := url.Values{}
	queryParams.Add("transport", "webSockets")

	negotiateURL := url.URL{
		Scheme:   "https",
		Host:     "dev.azure.com",
		Path:     fmt.Sprintf("%s/_apis/%s/signalr/negotiate", organization, projectID),
		RawQuery: queryParams.Encode(),
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", negotiateURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to negotiate connection token: %s", resp.Status)
	}

	var result negotiateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.ConnectionToken, nil
}

func (s *SignalRClient) Connect() error {
	s.logger.LogToFile("INFO", "connecting to signalr")
	signalrURL, header, err := GetConnectionParameters(s.Organization, s.AccountID, s.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get connection parameters for SignalR: %w", err)
	}
	c, _, err := websocket.DefaultDialer.Dial(signalrURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	err = sendHandshake(c)
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}
	s.Conn = c
	return nil
}

func (s *SignalRClient) ReadMessageWithRetry(attempts int, initialDelay time.Duration) ([]byte, error) {
	var message []byte
	var err error
	delay := initialDelay
	for i := 0; ; i++ {
		_, message, err = s.Conn.ReadMessage()
		if err == nil {
			return message, nil
		}
		if i >= (attempts - 1) {
			break
		}
		s.logger.LogToFile("ERROR", fmt.Sprintf("error reading message, retrying in %.2f: %v", delay.Seconds(), err))
		time.Sleep(delay)
		delay = delay*2 + time.Duration(rand.Int63n(int64(delay/2)))
	}
	return nil, err
}

// StartReceivingLoop starts the loop for receiving messages
func (s *SignalRClient) StartReceivingLoop(logChan chan<- utils.LogMsg) {
	defer func() {
		if err := s.Conn.Close(); err != nil {
			s.logger.LogToFile("ERROR", fmt.Sprintf("error closing connection: %v", err))
		} else {
			s.logger.LogToFile("INFO", "connection closed")
		}
	}()
receiveMessages:
	for {
		message, err := s.ReadMessageWithRetry(5, 1*time.Second)
		if err != nil {
			s.logger.LogToFile("ERROR", fmt.Sprintf("error reading message: %v", err))
			logChan <- utils.LogMsg{
				ReadLogError: err,
			}
			break
		}

		var response SignalRResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			s.logger.LogToFile("WARN", fmt.Sprintf("error unmarshalling message %s: %v", message, err))
			continue
		}

		for _, msg := range response.M {
			for _, detail := range msg.A {
				if len(detail.Lines) == 0 {
					logChan <- utils.LogMsg{
						BuildStatus: detail.Build.Status,
						BuildResult: detail.Build.Result,
					}
					if detail.Build.Status == "completed" {
						s.logger.LogToFile("INFO", "build completed")
						break receiveMessages
					} else {
						continue
					}
				}
				for _, line := range detail.Lines {
					logChan <- utils.LogMsg{
						NewContent:       line,
						TimelineRecordId: utils.TimelineRecordId(detail.TimelineRecordID),
						StepRecordId:     utils.StepRecordId(detail.StepRecordID),
						BuildStatus:      detail.Build.Status,
					}
				}
			}
		}
	}
}

// SendMessage sends a message to the SignalR connection
func (s *SignalRClient) SendMessage(hubName, methodName string, args []interface{}) error {
	s.logger.LogToFile("INFO", fmt.Sprintf("sending message to hub %s, method %s, args %v", hubName, methodName, args))
	message := map[string]interface{}{
		"H": hubName,
		"M": methodName,
		"A": args,
		"I": 0, // Message ID, can be incremented if needed
	}
	return s.Conn.WriteJSON(message)
}
