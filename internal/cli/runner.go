package cli

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type commandRunner interface {
	Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type systemRunner struct{}

func (systemRunner) Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	commandName, commandArgs, err := prepareCommand(name, args)
	if err != nil {
		return err
	}
	cmd := exec.Command(commandName, commandArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", strings.Join(append([]string{name}, args...), " "), err)
	}
	return nil
}

func prepareCommand(name string, args []string) (string, []string, error) {
	if runtime.GOOS != "windows" {
		return name, args, nil
	}

	resolved, err := exec.LookPath(name)
	if err != nil {
		return "", nil, fmt.Errorf("resolve %s: %w", name, err)
	}
	ext := strings.ToLower(filepath.Ext(resolved))
	if ext == ".cmd" || ext == ".bat" {
		return "cmd.exe", append([]string{"/c", resolved}, args...), nil
	}
	return resolved, args, nil
}
