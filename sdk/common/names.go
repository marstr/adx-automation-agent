package common

import (
	"io/ioutil"
	"strings"
)

// Defines well-known names in the A01 system
const (
	StorageVolumeNameArtifacts = "artifacts-storage"
	StorageVolumeNameSecrets   = "secrets-storage"
	StorageVolumeNameTools     = "tools-storage"
	DNSNameTaskStore           = "data-store-svc"
	DNSNameEmailService        = "email-report-svc"
	DNSNameReportService       = "report-internal-svc"
	SecretNameAgents           = "agent-secrets"
	SystemConfigMapName        = "a01-system-config"
)

const (
	// RunStatusInitialized is set when a run is just created
	RunStatusInitialized = "Initialized"

	// RunStatusPublished is set when tasks are added to the task broker queue
	RunStatusPublished = "Published"

	// RunStatusRunning is set when test job is created and start running
	RunStatusRunning = "Running"

	// RunStatusCompleted is set when all tasks are accomplished
	RunStatusCompleted = "Completed"
)

// Defines well-known keys in the a01 system config
const (
	ConfigKeyEndpointTaskBroker    = "endpoint.taskbroker"
	ConfigKeyUsernameTaskBroker    = "username.taskbroker"
	ConfigKeyPasswordKeyTaskBroker = "password.taskbroker"
	ConfigKeySecretTaskBroker      = "secret.taskbroker"
)

// Defines well-known keys in a product specific secret
const (
	ProductSecretKeyLogPathTemplate = "log.path.template"
)

// GetCurrentNamespace returns the namespace this Pod belongs to. Otherwise, it returns the empty string and a non-nil
// error.
func GetCurrentNamespace() (string, error) {
	contents, err := ioutil.ReadFile(PathKubeNamespace)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
