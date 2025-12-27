package task_api_client

import (
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/port"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// Client - клиент для task-service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	traceID := contextkeys.TraceIDFromContext(ctx)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// CreateTask создает новую задачу и возвращает ее ID
func (c *Client) CreateTask(ctx context.Context, name, taskType string, userID uuid.UUID, params ...any) (uuid.UUID, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "TaskApiClient",
		"method":    "CreateTask",
	})

	req := createTaskRequest{
		Name:            name,
		Type:            taskType,
		CreatedByUserID: userID.String(),
	}
	if len(params) > 0 {
		req.ObjectID = params[0].(string)
	}

	reqBody, _ := json.Marshal(req)

	url := c.baseURL + "/api/v1/tasks"
	clientLogger.Debug("Sending request to create task", port.Fields{"url": url, "task_name": name, "task_request": req})

	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		clientLogger.Error("Failed to perform request to task-service", err, nil)
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("task service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from task-service", err, port.Fields{"status_code": resp.StatusCode})
		return uuid.Nil, err
	}

	var respBody createTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		clientLogger.Error("Failed to decode create task response", err, nil)
		return uuid.Nil, err
	}

	taskID, err := uuid.Parse(respBody.ID)
	if err != nil {
		return uuid.Nil, err
	}

	clientLogger.Info("Successfully created task", port.Fields{"task_id": taskID.String()})

	return taskID, nil
}

// UpdateTaskStatus обновляет статус задачи
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status string) error {

	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "TaskApiClient",
		"method":    "UpdateTaskStatus",
		"task_id":   taskID.String(),
	})

	reqBody, _ := json.Marshal(updateTaskRequest{Status: status})

	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.baseURL, taskID.String())
	clientLogger.Debug("Sending request to update task status", port.Fields{"url": url, "new_status": status})

	resp, err := c.doRequest(ctx, http.MethodPut, url, bytes.NewBuffer(reqBody))
	if err != nil {
		clientLogger.Error("Failed to perform request to update task status", err, nil)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// статус-код указывает на ошибку
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("task service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from task-service", err, port.Fields{"status_code": resp.StatusCode})
		return err
	}

	clientLogger.Info("Successfully updated task status", nil)

	return nil
}
