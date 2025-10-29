package frontend

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bjartek/aether/pkg/events"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v3/process"
)

// FrontendManager manages the frontend process
type FrontendManager struct {
	cmd       *exec.Cmd
	cmdString string
	logger    zerolog.Logger
	mu        sync.Mutex
	running   bool
	ports     []string
}

func (m *FrontendManager) collectListeningPorts(proc *process.Process, visited map[int32]struct{}, portSet map[string]struct{}) {
	if proc == nil {
		return
	}

	pid := proc.Pid
	if _, seen := visited[pid]; seen {
		return
	}
	visited[pid] = struct{}{}

	name, _ := proc.Name()

	conns, err := proc.Connections()
	if err != nil {
		m.logger.Debug().Err(err).Int32("pid", pid).Str("process", name).Msg("Failed to get process connections")
	} else {
		for _, conn := range conns {
			if conn.Status == "LISTEN" {
				port := fmt.Sprintf("%d", conn.Laddr.Port)
				portSet[port] = struct{}{}
			}
		}
	}

	children, err := proc.Children()
	if err != nil {
		m.logger.Debug().Err(err).Int32("pid", pid).Str("process", name).Msg("Failed to get child processes")
		return
	}

	for _, child := range children {
		m.collectListeningPorts(child, visited, portSet)
	}
}

func NewFrontendManager(cmd string, logger zerolog.Logger) *FrontendManager {
	return &FrontendManager{
		cmdString: cmd,
		logger:    logger,
	}
}

func (m *FrontendManager) Start(teaProgram *tea.Program) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.logger.Info().Str("command", m.cmdString).Msg("Starting frontend process")

	effectiveCmd := strings.Trim(m.cmdString, "\"'")
	parts := strings.Fields(effectiveCmd)
	if len(parts) == 0 {
		return fmt.Errorf("frontend command is empty")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	setupProcessGroup(cmd)

	if wd, err := os.Getwd(); err != nil {
		m.logger.Warn().Err(err).Msg("Failed to get working directory for frontend process; using shell default")
	} else {
		cmd.Dir = wd
	}

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to create stdout pipe")
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to create stderr pipe")
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		m.logger.Error().Err(err).Msg("Failed to start frontend process")
		return fmt.Errorf("start command: %w", err)
	}

	m.cmd = cmd
	m.running = true

	// Only log output, no port detection
	go m.scanOutput(stdout, "frontend-out")
	go m.scanOutput(stderr, "frontend-err")

	// Start port detection using PID
	go m.detectPorts(teaProgram, cmd.Process.Pid)

	// Monitor the process
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		m.running = false

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				errOutput := string(exitErr.Stderr)
				if len(errOutput) > 500 {
					errOutput = errOutput[:500] + "... [truncated]"
				} else if errOutput == "" {
					errOutput = "(no error output)"
				}

				m.logger.Error().
					Int("exit_code", exitErr.ExitCode()).
					Str("command", m.cmdString).
					Str("error_output", errOutput).
					Msg("Frontend process exited with error")
			} else {
				m.logger.Error().
					Err(err).
					Str("command", m.cmdString).
					Msg("Frontend process exited with error")
			}
		} else {
			m.logger.Info().Msg("Frontend process exited")
		}
	}()

	return nil
}

func (m *FrontendManager) detectPorts(teaProgram *tea.Program, pid int) {
	// Wait for process to start
	time.Sleep(1 * time.Second)

	// Check ports every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		// Get connections for the process tree (frontend command + children)
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			m.logger.Error().Err(err).Int("pid", pid).Msg("Failed to get frontend process")
			continue
		}

		portSet := make(map[string]struct{})
		visited := make(map[int32]struct{})
		m.collectListeningPorts(p, visited, portSet)

		var ports []string
		for port := range portSet {
			ports = append(ports, port)
		}
		sort.Strings(ports)

		var newPorts []string

		m.mu.Lock()
		if len(ports) > 0 {
			m.logger.Debug().Strs("ports", ports).Msg("Frontend process listening ports detected")
		}

		prevSet := make(map[string]struct{}, len(m.ports))
		for _, existing := range m.ports {
			prevSet[existing] = struct{}{}
		}

		for _, port := range ports {
			if _, seen := prevSet[port]; !seen {
				newPorts = append(newPorts, port)
			}
		}

		m.ports = ports
		m.mu.Unlock()

		if teaProgram != nil {
			for _, port := range newPorts {
				teaProgram.Send(events.FrontendPortMsg{Port: port})
			}
		}
	}
}

func (m *FrontendManager) scanOutput(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		m.logger.Info().Str("prefix", prefix).Msg(line)
	}
	if err := scanner.Err(); err != nil {
		m.logger.Error().Err(err).Msg("Error scanning frontend output")
	}
}

func (m *FrontendManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info().Msg("Stopping frontend process")
	err := stopProcess(m.cmd)

	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to stop frontend process")
		return err
	}
	m.running = false
	m.cmd = nil
	return nil
}

func (m *FrontendManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *FrontendManager) Ports() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ports
}
