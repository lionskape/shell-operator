package executor

import (
	"bufio"
	"os/exec"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	utils "github.com/flant/shell-operator/pkg/utils/labels"
)

var ExecutorLock = &sync.Mutex{}

func Run(cmd *exec.Cmd) error {
	ExecutorLock.Lock()
	defer ExecutorLock.Unlock()

	// TODO context: hook name, hook phase, hook binding
	log.Debugf("Executing command %q in %q dir", strings.Join(cmd.Args, " "), cmd.Dir)

	return cmd.Run()
}

func RunAndLogLines(cmd *exec.Cmd, logLabels map[string]string) error {
	ExecutorLock.Lock()
	defer ExecutorLock.Unlock()

	logEntry := log.WithFields(utils.LabelsToLogFields(logLabels))
	stdoutLogEntry := logEntry.WithField("output", "stdout")
	stderrLogEntry := logEntry.WithField("output", "stderr")

	logEntry.Debugf("Executing command %q in %q dir", strings.Join(cmd.Args, " "), cmd.Dir)

	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutLogEntry.Info(scanner.Text())
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrLogEntry.Info(scanner.Text())
		}
	}()

	wg.Wait()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func Output(cmd *exec.Cmd) (output []byte, err error) {
	ExecutorLock.Lock()
	defer ExecutorLock.Unlock()

	output, err = cmd.Output()
	return
}

func MakeCommand(dir string, entrypoint string, args []string, envs []string) *exec.Cmd {
	cmd := exec.Command(entrypoint, args...)
	cmd.Env = append(cmd.Env, envs...)
	cmd.Dir = dir
	return cmd
}
