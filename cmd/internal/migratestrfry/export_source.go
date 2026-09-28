package migratestrfry

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

const maxExportLineBytes = 16 * 1024 * 1024

type exportCommand interface {
	StdoutPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	Kill() error
}

type exportCommandStarter func(context.Context, string, ...string) exportCommand

type exportSource struct {
	database string
	binary   string
	since    int64
	start    exportCommandStarter
	tempDir  string
}

func newExportSource(options CLIOptions) *exportSource {
	return &exportSource{
		database: options.StrfryDB,
		binary:   options.StrfryBin,
		since:    options.Since,
		start: func(ctx context.Context, binary string, args ...string) exportCommand {
			return execCommand{Cmd: exec.CommandContext(ctx, binary, args...)}
		},
	}
}

func (s *exportSource) Stream(ctx context.Context, handle func(int, []byte) error) error {
	configPath, err := s.createTemporaryConfig()
	if err != nil {
		return err
	}
	defer os.Remove(configPath)

	args := []string{"--config", configPath, "export"}
	if s.since > 0 {
		args = append(args, "--since", strconv.FormatInt(s.since, 10))
	}

	command := s.start(ctx, s.binary, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("strfry export stdout: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start strfry export: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), maxExportLineBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := handle(lineNumber, line); err != nil {
			_ = command.Kill()
			_ = command.Wait()
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		_ = command.Kill()
		_ = command.Wait()
		return fmt.Errorf("read strfry export: %w", err)
	}
	if err := command.Wait(); err != nil {
		return fmt.Errorf("strfry export failed: %w", err)
	}

	return nil
}

func (s *exportSource) createTemporaryConfig() (string, error) {
	file, err := os.CreateTemp(s.tempDir, "nrserver-strfry-*.conf")
	if err != nil {
		return "", fmt.Errorf("create temporary strfry config: %w", err)
	}
	path := file.Name()
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("secure temporary strfry config: %w", err)
	}
	if _, err := fmt.Fprintf(file, "db = %s\n", strconv.Quote(s.database)); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("write temporary strfry config: %w", err)
	}
	if err := file.Close(); err != nil {
		file = nil
		_ = os.Remove(path)
		return "", fmt.Errorf("close temporary strfry config: %w", err)
	}
	file = nil

	return path, nil
}

type execCommand struct {
	*exec.Cmd
}

func (c execCommand) Kill() error {
	if c.Process == nil {
		return nil
	}
	return c.Process.Kill()
}
