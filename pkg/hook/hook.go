package hook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/kennygrant/sanitize"

	"github.com/flant/shell-operator/pkg/executor"
)

type Hook struct {
	Name   string // The unique name like '002-prometheus-hooks/startup_hook'.
	Path   string // The absolute path to the executable file.
	Config *HookConfig

	hookManager HookManager
}

func NewHook(name, path string) *Hook {
	return &Hook{
		Name:   name,
		Path:   path,
		Config: &HookConfig{},
	}
}

func (h *Hook) WithHookManager(hookManager HookManager) {
	h.hookManager = hookManager
}

func (h *Hook) WithConfig(configOutput []byte) (hook *Hook, err error) {
	err = h.Config.LoadAndValidate(configOutput)
	if err != nil {
		return h, fmt.Errorf("load hook '%s' config: %s\nhook --config output: %s", h.Name, err.Error(), configOutput)
	}

	return h, nil
}

func (h *Hook) Run(bindingType BindingType, context []BindingContext, logLabels map[string]string) error {
	var versionedContext = ConvertBindingContextList(h.Config.Version, context)

	contextPath, err := h.prepareBindingContextJsonFile(versionedContext)
	if err != nil {
		return err
	}

	envs := []string{}
	envs = append(envs, os.Environ()...)
	if contextPath != "" {
		envs = append(envs, fmt.Sprintf("BINDING_CONTEXT_PATH=%s", contextPath))
	}

	hookCmd := executor.MakeCommand(path.Dir(h.Path), h.Path, []string{}, envs)

	err = executor.RunAndLogLines(hookCmd, logLabels)
	if err != nil {
		return fmt.Errorf("%s FAILED: %s", h.Name, err)
	}

	return nil
}

func (h *Hook) SafeName() string {
	return sanitize.BaseName(h.Name)
}

func (h *Hook) prepareBindingContextJsonFile(context interface{}) (string, error) {
	data, _ := json.MarshalIndent(context, "", "  ")
	bindingContextPath := filepath.Join(h.hookManager.TempDir(), fmt.Sprintf("hook-%s-binding-context.json", h.SafeName()))

	err := ioutil.WriteFile(bindingContextPath, data, 0644)
	if err != nil {
		return "", err
	}

	return bindingContextPath, nil
}
