package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/adx-automation-agent/sdk/common"
	"github.com/Azure/adx-automation-agent/sdk/kubeutils"
	"github.com/Azure/adx-automation-agent/sdk/models"
	"github.com/Azure/adx-automation-agent/sdk/schedule"
	"github.com/sirupsen/logrus"
)

type (
	// ErrNoExecutable is returned when a file that was expected to be executed is not found.
	ErrNoExecutable string
)

func (e ErrNoExecutable) Error() string {
	return fmt.Sprintf("did not find executable %s", string(e))
}

// metadata about this executable. These values should be populated using the flag:
//     -ldflags "-X github.com/Azure/adx-automation-agents/main.version={your value}"`
var (
	version      string
	sourceCommit string
)

var (
	podName         = os.Getenv(common.EnvPodName)
	logPathTemplate = ""
)

func init() {
	if version == "" {
		version = "Unknown"
	}

	if sourceCommit == "" {
		sourceCommit = "Unknown"
	}
}

func ckEnvironment() error {
	required := []string{common.EnvKeyInternalCommunicationKey, common.EnvJobName}

	missing := make([]string, 0, len(required))

	for _, r := range required {
		if _, ok := os.LookupEnv(r); !ok {
			missing = append(missing, r)
		}
	}

	if len(missing) > 0 {
		builder := &bytes.Buffer{}
		builder.WriteString("the following required environment variables are missing:\n")
		for _, entry := range missing {
			builder.WriteRune('\t')
			builder.WriteString(entry)
			builder.WriteRune('\n')
		}

		return errors.New(builder.String())
	}

	return nil
}

func main() {
	ctx := context.Background()
	logger := logrus.StandardLogger()

	logger.Infof("A01 Droid Engine\n\tVersion: %s\n\tCommit: %s", version, sourceCommit)

	err := ckEnvironment()
	if err != nil {
		logger.Fatal(err)
	}

	jobName := os.Getenv(common.EnvJobName)
	productName, runID, err := splitJobName(jobName)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Run ID: %s", runID)

	taskBroker, err := schedule.CreateInClusterTaskBroker()
	if err != nil {
		logger.Fatal(err)
	}

	queue, ch, err := taskBroker.QueueDeclare(jobName)
	if err != nil {
		logger.Fatal("Failed to connect to the task broker.")
	}

	if bLogPathTemplate, exists := kubeutils.TryGetSecretInBytes(
		productName,
		common.ProductSecretKeyLogPathTemplate); exists {
		logPathTemplate = string(bLogPathTemplate)
	}

	// If pod prep fails, preparePod will terminate the program
	preparePod(ctx, logrus.StandardLogger(), common.PathScriptPreparePod)

	for {
		delivery, ok, err := ch.Get(queue.Name, false /* autoAck*/)
		if err != nil {
			logger.Fatal("Failed to get a delivery: ", err)
		}

		if !ok {
			logger.Info("No more task in the queue. Exiting successfully.")
			break
		}

		var output []byte
		var taskResult *models.TaskResult
		var setting models.TaskSetting
		err = json.Unmarshal(delivery.Body, &setting)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to unmarshel a delivery's body in JSON: %s", err.Error())
			logrus.Error(errorMsg)

			taskResult = setting.CreateUncompletedTask(podName, runID, errorMsg)
		} else {
			logrus.Infof("Run task %s", setting.GetIdentifier())

			result, duration, executeOutput := setting.Execute()
			taskResult = setting.CreateCompletedTask(result, duration, podName, runID)
			output = executeOutput
		}

		taskResult, err = taskResult.CommitNew()
		if err != nil {
			logrus.Errorf("Failed to commit a new task: %v", err)
		} else {
			taskLogPath, err := taskResult.SaveTaskLog(output)
			if err != nil {
				logrus.Error(err)
			}

			afterTask(ctx, logger, common.PathScriptAfterTest, taskResult)

			if len(logPathTemplate) > 0 {
				taskResult.ResultDetails[common.KeyTaskLogPath] = strings.Replace(
					logPathTemplate,
					"{}",
					taskLogPath,
					1)

				taskResult.ResultDetails[common.KeyTaskRecordPath] = strings.Replace(
					logPathTemplate,
					"{}",
					path.Join(strconv.Itoa(taskResult.RunID), fmt.Sprintf("recording_%d.yaml", taskResult.ID)),
					1)

				_, err := taskResult.CommitChanges()
				if err != nil {
					logrus.Error(err)
				}
			}
		}

		err = delivery.Ack(false)
		if err != nil {
			logrus.Errorf("Failed to ack delivery: %v", err)
		} else {
			logrus.Info("ACK")
		}
	}
}

// splitJobName breaks down a jonName string adhering to a known format into its composing elements.
//
// Note: It is structured as a variable instead of a normal function in order to allow closure to essentially create a
// locally scoped regular expression that gets compiled exactly once at program initialization time instead of panicking
// only when a program needs a jobID for the first time. This should make literally any integration test fail if this
// becomes an invalid regular expression.
var splitJobName = func() func(string) (string, string, error) {
	jobNamePattern := regexp.MustCompile(`^(?P<product>.+)-(?P<runID>.+)-(?P<randomID>.+)$`)

	return func(jobName string) (productName, runID string, err error) {
		results := jobNamePattern.FindStringSubmatch(jobName)
		if results == nil {
			err = fmt.Errorf("%q is not in format <product>-<runID>-<randomID>", jobName)
			return
		}

		productName, runID = results[1], results[2]
		return
	}
}()

// preparePod executes a file while logging. The language used for logging assumes that this will be executed exactly
// once as an initialization step before entering the task loop.
func preparePod(ctx context.Context, logger *logrus.Logger, prepPath string) {
	_, err := os.Stat(prepPath)
	if os.IsNotExist(err) {
		logger.Infof("skipping pod preparation, %s not present", prepPath)
	}

	output := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, prepPath)
	cmd.Stdout = output
	cmd.Stderr = output
	err = cmd.Run()

	if err == nil {
		const affirmative = "pod prepared"
		if output.Len() > 0 {
			logger.Info(affirmative + " output:\n" + output.String())
		} else {
			logger.Info(affirmative)
		}
	} else {
		const negative = "pod preparation failed\n\terr: %v"
		if output.Len() > 0 {
			logger.Fatalf(negative+"\n\toutput:\n%s", err, output.String())
		} else {
			logger.Fatalf(negative, err)
		}
	}
}

func afterTask(ctx context.Context, logger *logrus.Logger, afterPath string, taskResult *models.TaskResult) {
	_, err := os.Stat(afterPath)
	if os.IsNotExist(err) {
		logger.Info("no after task action found")
		// Missing after task executable is not considered an error.
		return
	}

	logrus.Infof("Executing after task %s.", common.PathScriptAfterTest)

	taskInBytes, err := json.Marshal(taskResult)
	if err != nil {
		logger.Errorf("unable to encode task to JSON: %s", err.Error())
		return
	}

	outBuf := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, afterPath, common.PathMountArtifacts, string(taskInBytes))
	cmd.Stdout = outBuf
	cmd.Stderr = outBuf

	err = cmd.Run()
	if err == nil {
		const affirmative = "after-task succeeded"
		if outBuf.Len() > 0 {
			logger.Info(affirmative + " output:\n" + outBuf.String())
		} else {
			logger.Info(affirmative)
		}
	} else {
		const negative = "after-task failed\n\terr: %v"
		if outBuf.Len() > 0 {
			logger.Errorf(negative+"\n\toutput:\n%s", err, outBuf.String())
		} else {
			logger.Errorf(negative, err)
		}
	}
}
