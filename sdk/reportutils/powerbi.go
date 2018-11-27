package reportutils

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Azure/adx-automation-agent/sdk/common"
	"github.com/Azure/adx-automation-agent/sdk/models"
	"github.com/sirupsen/logrus"
)

// RefreshPowerBI requests the PowerBI service to refresh a data set
func RefreshPowerBI(logger *logrus.Logger, run *models.Run, product string) {
	const reportEndpoint = "http://" + common.DNSNameReportService + "/report"

	if !run.IsOfficial() {
		logger.Info("skip PowerBI refresh: run is not official")
		return
	}

	content := map[string]interface{}{
		"product": product,
		"runID":   run.ID,
	}
	body, err := json.Marshal(content)
	if err != nil {
		logger.Errorf("failed to marshal JSON before refreshing PowerBI: %v", err)
		return
	}

	req, err := http.NewRequest(
		http.MethodPost,
		reportEndpoint,
		bytes.NewBuffer(body))
	if err != nil {
		logger.Errorf("fail to create request to refresh PowerBI: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Errorf("Fail to send request to PowerBI service: %v", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		logger.Info("PowerBI refresh requested")
	} else {
		logger.Errorf("unexpected response code while sending report: %d", resp.StatusCode)
	}
}
