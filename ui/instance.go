package ui

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SingleInstance keeps one copy of the application running per user: a
// second start passes its arguments (files to open, mostly) to the running
// copy, which shows its window, and quits.
//
//	instance, ok := ui.AcquireSingleInstance("myapp", os.Args[1:])
//	if !ok {
//		return // the running copy shows itself
//	}
//	defer instance.Close()
//	instance.OnActivate(func(args []string) {
//		form.Activate()
//		openFiles(args)
//	})
//
// The copies talk through a local socket next to a lock file (in the
// application's settings directory); the system drops the lock of a copy
// that crashed.
type SingleInstance struct {
	lock     *os.File
	listener net.Listener
}

const (
	instanceShowCommand = "show"
	instanceArgCommand  = "arg "
	// instanceDialTimeout is how long a second start tries to reach the
	// running copy, which may be starting up itself
	instanceDialTimeout = 3 * time.Second
	// instanceMaxMessage caps what the running copy reads from one
	// connection: the arguments of a start with room to spare, and no more
	// memory for whatever else writes to the socket
	instanceMaxMessage = 16 << 20
	// instanceIOTimeout is how long one side waits for the whole message
	// of a start
	instanceIOTimeout = 2 * time.Second
)

// AcquireSingleInstance makes this process the running copy of the
// application. When another copy runs, it is given the arguments and asked
// to show its window, and ok is false: this one must quit. When the lock
// can't be taken at all (e.g. no access to the directory), the application
// runs unprotected rather than not at all.
func AcquireSingleInstance(app string, args []string) (instance *SingleInstance, ok bool) {
	dir, err := SettingsDir(app)
	if err != nil {
		return &SingleInstance{}, true
	}
	return acquireSingleInstance(dir, args)
}

func acquireSingleInstance(dir string, args []string) (*SingleInstance, bool) {
	c := &SingleInstance{}
	lock, err := os.OpenFile(filepath.Join(dir, "instance.lock"), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return c, true
	}
	if !tryLockFile(lock) {
		lock.Close()
		if socket := instanceSocketPath(dir); socket != "" {
			askToShow(socket, args)
		}
		return nil, false
	}
	c.lock = lock
	socket := instanceSocketPath(dir)
	if socket == "" {
		return c, true // no private place for it: a second start can't show this copy
	}
	// A socket left by a copy that crashed: the lock shows no one uses it
	os.Remove(socket)
	if c.listener, err = net.Listen("unix", socket); err != nil {
		c.listener = nil // still the only copy, a second start just can't show it
	}
	return c, true
}

// OnActivate calls f on the UI thread each time the application is started
// again, with the arguments of that start
func (c *SingleInstance) OnActivate(f func(args []string)) {
	if c == nil || c.listener == nil {
		return
	}
	// The goroutine keeps its own copy: Close sets the field to nil from
	// the UI thread while the goroutine may still be handling a connection
	ln := c.listener
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // closed
			}
			if args, ok := readInstanceMessage(conn); ok {
				Invoke(func() { f(args) })
			}
			conn.Close()
		}
	}()
}

// readInstanceMessage reads the message of a second start: one line
// "arg <quoted>" per argument, then "show". ok is false for a message that
// is cut short, too long or holds anything else: the start is ignored
// rather than delivered with arguments it did not send.
func readInstanceMessage(conn net.Conn) (args []string, ok bool) {
	conn.SetReadDeadline(time.Now().Add(instanceIOTimeout))
	sc := bufio.NewScanner(io.LimitReader(conn, instanceMaxMessage))
	sc.Buffer(make([]byte, 0, 4096), instanceMaxMessage)
	for sc.Scan() {
		line := sc.Text()
		if quoted, isArg := strings.CutPrefix(line, instanceArgCommand); isArg {
			arg, err := strconv.Unquote(quoted)
			if err != nil {
				return nil, false
			}
			args = append(args, arg)
			continue
		}
		return args, line == instanceShowCommand
	}
	return nil, false
}

// writeInstanceMessage gives the message readInstanceMessage reads. Each
// argument is quoted, so a line break in it (legal in a Unix file name)
// can't end it early or pass for a command.
func writeInstanceMessage(args []string) []byte {
	var msg strings.Builder
	for _, arg := range args {
		msg.WriteString(instanceArgCommand + strconv.Quote(arg) + "\n")
	}
	msg.WriteString(instanceShowCommand + "\n")
	return []byte(msg.String())
}

// Close releases the lock: another start becomes the running copy
func (c *SingleInstance) Close() {
	if c == nil {
		return
	}
	if c.listener != nil {
		c.listener.Close() // removes the socket file
		c.listener = nil
	}
	if c.lock != nil {
		// The lock file stays: removing it could let two copies lock
		// different files
		c.lock.Close()
		c.lock = nil
	}
}

// maxSocketPath is the longest unix socket path every system takes (104
// bytes with the terminating zero on macOS, 108 on Linux)
const maxSocketPath = 103

// instanceSocketPath returns where the running copy of the directory
// listens, "" if there is no place only the user can write to. A unix
// socket path is limited to about 100 bytes, which a long home path can
// exceed, so the short runtime directory is used when there is one (Linux),
// and a private temporary directory of the user when the path in the
// settings directory is too long (macOS: ~/Library/Application Support/...).
// Not the shared /tmp itself: another user could take the name first and
// get the arguments of every later start.
func instanceSocketPath(dir string) string {
	sum := sha256.Sum256([]byte(dir))
	name := "nui-" + hex.EncodeToString(sum[:8]) + ".sock"
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, name)
	}
	if path := filepath.Join(dir, "instance.sock"); len(path) <= maxSocketPath {
		return path
	}
	tmp, err := privateTempDir()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, name)
}

// askToShow gives the arguments to the running copy and asks it to show its
// window
func askToShow(socketPath string, args []string) error {
	// Let the running copy take the focus, which Windows gives only to the
	// foreground process
	allowSetForeground()
	deadline := time.Now().Add(instanceDialTimeout)
	for {
		conn, err := net.DialTimeout("unix", socketPath, time.Second)
		if err == nil {
			defer conn.Close()
			conn.SetWriteDeadline(time.Now().Add(instanceIOTimeout))
			_, err = conn.Write(writeInstanceMessage(args))
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}
