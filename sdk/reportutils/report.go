package reportutils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/adx-automation-agent/sdk/common"
	"github.com/Azure/adx-automation-agent/sdk/models"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{}

// Report method requests the email service to send emails
func Report(logger *logrus.Logger, run *models.Run, receivers []string, templateURL string) {
	const reportEndpoint = "http://" + common.DNSNameEmailService + "/report"

	// Emails should not be sent to all the team if the run was not set with a remark
	// Only acceptable remark for sending emails to whole team is 'official'
	if !run.IsOfficial() {
		receivers = []string{}
	}

	if email, ok := run.Settings[common.KeyUserEmail]; ok {
		receivers = append(receivers, email.(string))
	}

	if len(receivers) > 0 {
		content := make(map[string]string)
		content["run_id"] = strconv.Itoa(run.ID)
		content["receivers"] = strings.Join(receivers, ",")
		content["template"] = templateURL

		body, err := json.Marshal(content)
		if err != nil {
			logger.Errorf("fail to marshal JSON during request sending email: %v", err)
			return
		}

		logger.Debugf("report message body:\n%s", body)
		req, err := http.NewRequest(
			http.MethodPost,
			reportEndpoint,
			bytes.NewBuffer(body))
		if err != nil {
			logger.Errorf("fail to create request to requesting email: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("fail to send request to email service: ", err)
			return
		}

		switch resp.StatusCode {
		case http.StatusOK:
			logger.Info("report sent")
		default:
			logger.Errorf("unexpected response code while sending report: %d", resp.StatusCode)
		}
	} else {
		logger.Warn("no recipients to send the report")
	}
}
