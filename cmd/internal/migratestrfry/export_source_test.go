package migratestrfry

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportSourceStreamPassesExpectedArgvAndRemovesConfig(t *testing.T) {
	t.Parallel()

	command := &fakeExportCommand{stdout: io.NopCloser(strings.NewReader("one\n\ntwo\n"))}
	var binary string
	var args []string
	source := &exportSource{
		database: "/source/db\"quoted",
		binary:   "/usr/bin/strfry",
		since:    123,
		tempDir:  t.TempDir(),
		start: func(_ context.Context, gotBinary string, gotArgs ...string) exportCommand {
			binary = gotBinary
			args = append([]string{}, gotArgs...)
			return command
		},
	}

	lines := []string{}
	err := source.Stream(context.Background(), func(_ int, line []byte) error {
		lines = append(lines, string(line))
		return nil
	})

	require.NoError(t, err)
	require.True(t, command.started)
	require.Equal(t, "/usr/bin/strfry", binary)
	require.Equal(t, []string{"--config", args[1], "export", "--since", "123"}, args)
	require.Equal(t, []string{"one", "two"}, lines)
	require.NoFileExists(t, args[1])
}

func TestExportSourceStreamReturnsChildFailure(t *testing.T) {
	t.Parallel()

	source := &exportSource{
		database: "db",
		binary:   "strfry",
		tempDir:  t.TempDir(),
		start: func(context.Context, string, ...string) exportCommand {
			return &fakeExportCommand{
				stdout:  io.NopCloser(strings.NewReader("")),
				waitErr: errors.New("exit status 2"),
			}
		},
	}

	err := source.Stream(context.Background(), func(int, []byte) error { return nil })
	require.ErrorContains(t, err, "strfry export failed")
}

func TestExportSourceTemporaryConfigContainsOnlyEscapedDatabase(t *testing.T) {
	t.Parallel()

	source := &exportSource{database: "/source/db\"quoted", tempDir: t.TempDir()}
	path, err := source.createTemporaryConfig()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(path) })

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "db = \"/source/db\\\"quoted\"\n", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

type fakeExportCommand struct {
	stdout   io.ReadCloser
	startErr error
	waitErr  error
	started  bool
	killed   bool
}

func (c *fakeExportCommand) StdoutPipe() (io.ReadCloser, error) {
	return c.stdout, nil
}

func (c *fakeExportCommand) Start() error {
	c.started = true
	return c.startErr
}

func (c *fakeExportCommand) Wait() error {
	return c.waitErr
}

func (c *fakeExportCommand) Kill() error {
	c.killed = true
	return nil
}
