/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testserver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// The etcd server lives in its own module, hack/etcd-testserver, so that
// go.etcd.io/etcd/server/v3 stays out of the Kubernetes module graph. The
// binary is found through KUBE_ETCD_TESTSERVER, next to the test binary,
// under <repo>/_output/local/bin, or built there on first use.
const (
	envVar     = "KUBE_ETCD_TESTSERVER"
	moduleDir  = "hack/etcd-testserver"
	binaryName = "etcd-testserver"
)

var (
	binaryOnce sync.Once
	binaryFile string
	binaryErr  error
)

func binaryPath(t testing.TB) string {
	t.Helper()
	binaryOnce.Do(func() { binaryFile, binaryErr = resolveBinary() })
	if binaryErr != nil {
		t.Fatalf("%s: %v", binaryName, binaryErr)
	}
	return binaryFile
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func resolveBinary() (string, error) {
	if p := os.Getenv(envVar); p != "" {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), exeName()); fileExists(p) {
			return p, nil
		}
	}
	root, err := repoRoot()
	if err != nil {
		return "", fmt.Errorf("%w; set %s to a binary built from %s", err, envVar, moduleDir)
	}
	return buildBinary(root)
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, moduleDir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("kubernetes repository root not found")
		}
		dir = parent
	}
}

// sourceHash changes whenever the helper's module files, platform or Go
// version change. hack/build-etcd-testserver.sh computes the same value.
func sourceHash(src string) (string, error) {
	h := sha256.New()
	for _, name := range []string{"go.mod", "go.sum", "main.go"} {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return "", err
		}
		h.Write(b)
	}
	fmt.Fprintf(h, "%s/%s\n%s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	return hex.EncodeToString(h.Sum(nil)), nil
}

func goTool() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	return filepath.Join(runtime.GOROOT(), "bin", "go")
}

func buildBinary(root string) (string, error) {
	src := filepath.Join(root, moduleDir)
	hash, err := sourceHash(src)
	if err != nil {
		return "", err
	}
	out := filepath.Join(root, "_output", "local", "bin", exeName())
	stamp := out + ".srchash"
	upToDate := func() bool {
		b, err := os.ReadFile(stamp)
		return err == nil && string(b) == hash && fileExists(out)
	}
	if upToDate() {
		return out, nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return "", err
	}
	unlock, err := lockFile(out + ".lock")
	if err != nil {
		return "", err
	}
	defer unlock()
	if upToDate() {
		return out, nil
	}
	tmp := out + ".tmp"
	cmd := exec.Command(goTool(), "build", "-o", tmp, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "CGO_ENABLED=0")
	if msg, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build %s: %w\n%s", moduleDir, err, msg)
	}
	if err := os.Rename(tmp, out); err != nil {
		return "", err
	}
	return out, os.WriteFile(stamp, []byte(hash), 0644)
}

// lockFile serialises builds across test binaries running in parallel.
func lockFile(path string) (func(), error) {
	const stale = 10 * time.Minute
	deadline := time.Now().Add(stale)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) > stale {
			os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for %s", path)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
