package ui

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
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
		askToShow(instanceSocketPath(dir), args)
		return nil, false
	}
	c.lock = lock
	// A socket left by a copy that crashed: the lock shows no one uses it
	socket := instanceSocketPath(dir)
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
	go func() {
		for {
			conn, err := c.listener.Accept()
			if err != nil {
				return // closed
			}
			conn.SetReadDeadline(time.Now().Add(time.Second))
			var args []string
			sc := bufio.NewScanner(conn)
			for sc.Scan() {
				line := sc.Text()
				if arg, ok := strings.CutPrefix(line, instanceArgCommand); ok {
					args = append(args, arg)
					continue
				}
				if strings.TrimSpace(line) == instanceShowCommand {
					Invoke(func() { f(args) })
				}
				break
			}
			conn.Close()
		}
	}()
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

// instanceSocketPath returns where the running copy of the directory
// listens. A unix socket path is limited to about 100 bytes, which a long
// home path can exceed, so the short runtime directory is used when there
// is one (Linux).
func instanceSocketPath(dir string) string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		sum := sha256.Sum256([]byte(dir))
		return filepath.Join(runtimeDir, "nui-"+hex.EncodeToString(sum[:8])+".sock")
	}
	return filepath.Join(dir, "instance.sock")
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
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			var msg strings.Builder
			for _, arg := range args {
				msg.WriteString(instanceArgCommand + arg + "\n")
			}
			msg.WriteString(instanceShowCommand + "\n")
			_, err = conn.Write([]byte(msg.String()))
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}
