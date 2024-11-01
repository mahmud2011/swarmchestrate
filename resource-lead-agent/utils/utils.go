package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/config"
	"gopkg.in/yaml.v3"
)

func YAMLtoJSON(r *http.Request, fieldName string) (json.RawMessage, error) {
	file, _, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var jsonData []map[string]interface{}
	decoder := yaml.NewDecoder(io.NopCloser(io.MultiReader(bytes.NewReader(fileBytes))))

	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		jsonData = append(jsonData, doc)
	}

	return json.Marshal(jsonData)
}

func FilterEligibleNodes(nodes []*domain.Node) []*domain.Node {
	var eligibleNodes []*domain.Node
	now := time.Now().UTC()

	for _, node := range nodes {
		if node.IsReady &&
			node.IsSchedulable &&
			!node.IsDiskPressureExists &&
			!node.IsMemoryPressureExists &&
			!node.IsPIDPressureExists &&
			!node.UpdatedAt.Before(now.Add(-1*config.Get().App.NodeHeartbeatTimeout)) {
			eligibleNodes = append(eligibleNodes, node)
		}
	}

	return eligibleNodes
}

func ForwardReq(host string, r *http.Request) (*http.Response, error) {
	newURL := fmt.Sprintf("http://%s:%d", host, 80) + r.URL.Path

	proxyReq, err := http.NewRequest(r.Method, newURL, r.Body)
	if err != nil {
		return nil, err
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func StringToPointer(s string) *string {
	return &s
}
