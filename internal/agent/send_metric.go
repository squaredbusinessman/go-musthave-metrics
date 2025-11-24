package agent

import (
	"fmt"
	"net/http"
)

// sendMetric Функция отправки ОДНОЙ метрики
func sendMetric(client *http.Client, serverAddr string, metricType string, metricName string, metricValue string) error {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", serverAddr, metricType, metricName, metricValue)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	return nil
}
