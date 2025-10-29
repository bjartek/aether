package frontend

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

type FrontendProcess struct {
	command string
	cmd     *exec.Cmd
	logger  zerolog.Logger
	status  string
}

func NewFrontendProcess(command string, logger zerolog.Logger) *FrontendProcess {
	return &FrontendProcess{
		command: command,
		logger:  logger.With().Str("component", "frontend").Logger(),
		status:  "stopped",
	}
}

func (fp *FrontendProcess) Start(ctx context.Context) error {
	// Split the command string into command and args
	parts := strings.Fields(fp.command)
	if len(parts) == 0 {
		return nil
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	fp.cmd = cmd
	fp.status = "running"

	// Capture stdout and stderr
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fp.logger.Info().Msg(scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fp.logger.Error().Msg(scanner.Text())
		}
	}()

	// Wait for the command to exit in the background
	go func() {
		err := cmd.Wait()
		if err != nil {
			fp.logger.Error().Err(err).Msg("frontend process exited with error")
		} else {
			fp.logger.Info().Msg("frontend process exited")
		}
		fp.status = "exited"
	}()

	return nil
}

func (fp *FrontendProcess) Stop() {
	if fp.cmd != nil && fp.cmd.Process != nil {
		_ = fp.cmd.Process.Kill()
	}
	fp.status = "stopped"
}

func (fp *FrontendProcess) Status() string {
	return fp.status
}
