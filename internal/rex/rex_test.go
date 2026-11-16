//go:build darwin || linux || windows

package rex

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSecurityControls(t *testing.T) {
	// 1. sanitizeTitle strips C0/C1 control characters and escape sequences.
	dirty := "my-title\x07\x1b]52;c;dGVzdA==\x07\x1b[2J\x9c-clean"
	got := sanitizeTitle(dirty)
	if strings.ContainsAny(got, "\x07\x1b\x9c") || got != "my-title]52;c;dGVzdA==[2J-clean" {
		t.Fatalf("sanitizeTitle(%q) = %q", dirty, got)
	}

	// 2. readLine enforces maxBytes cap on oversized lines without newline.
	oversized := bufio.NewReader(strings.NewReader(strings.Repeat("a", 2048)))
	if _, err := readLine(oversized, 1024); err == nil {
		t.Fatal("readLine accepted line exceeding maxBytes")
	}
	valid := bufio.NewReader(strings.NewReader("{\"op\":\"hello\"}\ntrailing"))
	line, err := readLine(valid, 1024)
	if err != nil || string(line) != `{"op":"hello"}` {
		t.Fatalf("readLine valid = %q, %v", line, err)
	}

	// 3. SocketPath fallback uses a dedicated subdirectory rather than flat TempDir.
	longDir := filepath.Join(os.TempDir(), strings.Repeat("longdir-", 15))
	t.Setenv("GOREX_DIR", longDir)
	sp := SocketPath()
	if filepath.Dir(sp) == filepath.Clean(os.TempDir()) {
		t.Fatalf("SocketPath(%q) placed directly in TempDir", sp)
	}

	// 4. listenSocket refuses to remove a non-socket file or symlink at the socket path.
	tmp := t.TempDir()
	badSock := filepath.Join(tmp, "server.sock")
	if err := os.WriteFile(badSock, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ln, err := listenSocket(badSock); err == nil {
		ln.Close()
		t.Fatal("listenSocket overwrote a regular file")
	}
}

func TestServer(t *testing.T) {
	// A short path, for the socket: macOS's temporary directory is long.
	base, shell, work := "/tmp", "/bin/sh", "/tmp"
	// The command prints "hello" without the line typed showing it, and
	// runs a program for a while.
	typed, sleeper := "echo he''llo; sleep 1\n", "sleep"
	if runtime.GOOS == "windows" {
		base, shell, work = "", "cmd.exe", os.TempDir()
		typed, sleeper = "echo he^llo & ping -n 3 127.0.0.1 >nul\r", "ping"
	}
	dir, err := os.MkdirTemp(base, "rex")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("GOREX_DIR", dir)
	go Serve()
	var c *Client
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(filepath.Join(dir, "server.sock")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	c, err = Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Shutdown()
	h, err := c.Hello()
	if err != nil || h.Version != ProtocolVersion {
		t.Fatalf("hello %+v %v", h, err)
	}
	t.Logf("host %+v", h.Host)
	info, err := c.Create(CreateOptions{Command: []string{shell}, Dir: work, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	s := c.Stream(info.ID, 80, 24)
	s.Resize(100, 30)
	s.Write([]byte(typed))
	var out bytes.Buffer
	buf := make([]byte, 4096)
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "hello") && time.Now().Before(deadline) {
		n, err := s.Read(buf)
		out.Write(buf[:n])
		if err != nil {
			break
		}
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("output %q", out.String())
	}
	time.Sleep(200 * time.Millisecond)
	list, err := c.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list %+v %v", list, err)
	}
	in := list[0]
	t.Logf("info %+v", in)
	if !strings.EqualFold(in.Program, sleeper) || in.Idle || in.Cols != 100 {
		t.Errorf("info %+v", in)
	}
	if !strings.HasSuffix(strings.ToLower(in.Dir), strings.ToLower(work)) {
		t.Errorf("dir %q", in.Dir)
	}
	s.Close()
	// Attaching again replays what it printed.
	s2 := c.Stream(info.ID, 100, 30)
	s2.Resize(100, 30)
	n, _ := s2.Read(buf)
	if !strings.Contains(string(buf[:n]), "hello") {
		t.Errorf("replay %q", buf[:n])
	}
	if err := c.Kill(info.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := c.List(); len(list) != 0 {
		t.Errorf("sessions left %+v", list)
	}
}
