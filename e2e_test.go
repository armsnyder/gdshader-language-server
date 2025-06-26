package main_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/samber/lo"
)

func TestE2E(t *testing.T) {
	g := NewWithT(t)

	// Build the server binary
	binPath := filepath.Join(t.TempDir(), "gdshader-language-server")
	g.Expect(exec.Command("go", "build", "-cover", "-o", binPath, ".").Run()).To(Succeed(), "Failed to build server")

	// Configure the server command
	const coverDir = "tmp/cover/e2e"
	lo.Must0(os.MkdirAll(coverDir, 0o700))
	var stdout bytes.Buffer
	cmd := exec.Command(binPath, "-debug")
	cmd.Env = []string{"GOCOVERDIR=" + coverDir}
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout
	stdin := lo.Must(cmd.StdinPipe())

	// Start the server
	g.Expect(cmd.Start()).To(Succeed(), "Failed to start server")
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// Send JSON-RPC requests to the server

	send := func(s string) {
		var buf bytes.Buffer
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(s)))
		buf.WriteString("\r\n\r\n")
		buf.WriteString(s)
		lo.Must(io.Copy(stdin, &buf))
	}

	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	send(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`)
	send(`{"jsonrpc":"2.0","method":"exit"}`)

	// Wait for the server to exit
	select {
	case err := <-lo.Async(cmd.Wait):
		g.Expect(err).ToNot(HaveOccurred(), "Server exited with error")
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not exit in time")
	}

	// Check the output

	var expected string

	expect := func(s string) {
		expected += "Content-Length: " + strconv.Itoa(len(s)) + "\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n" + s
	}

	expect(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":{"openClose":true,"change":2},"completionProvider":{}},"serverInfo":{"name":"gdshader-language-server","version":"development"}}}`)
	expect(`{"jsonrpc":"2.0","id":2,"result":null}`)

	g.Expect(stdout.String()).To(BeComparableTo(string(expected)), "Output does not match expected")
}
