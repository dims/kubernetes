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
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/kubernetes"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"

	storagetesting "k8s.io/apiserver/pkg/storage/testing"
)

// Config is what a test may tweak before the server starts. It mirrors
// hack/etcd-testserver's Config; keep both in sync.
type Config struct {
	Dir                         string        `json:"dir"`
	ClientURL                   string        `json:"clientURL"`
	PeerURL                     string        `json:"peerURL"`
	QuotaBackendBytes           int64         `json:"quotaBackendBytes,omitempty"`
	WatchProgressNotifyInterval time.Duration `json:"watchProgressNotifyInterval,omitempty"`
	// Setting ClientCertFile serves and dials the client URL over TLS.
	ClientCertFile      string `json:"clientCertFile,omitempty"`
	ClientKeyFile       string `json:"clientKeyFile,omitempty"`
	ClientTrustedCAFile string `json:"clientTrustedCAFile,omitempty"`
}

type reply struct {
	ClientURLs []string `json:"clientURLs,omitempty"`
	Error      string   `json:"error,omitempty"`
	AddrInUse  bool     `json:"addrInUse,omitempty"`
}

// getAvailablePorts returns TCP ports that are available for binding.
func getAvailablePorts(count int) ([]int, error) {
	ports := []int{}
	for i := 0; i < count; i++ {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, fmt.Errorf("could not bind to a port: %v", err)
		}
		// It is possible but unlikely that someone else will bind this port before we get a chance to use it.
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return ports, nil
}

func newTestConfig(t testing.TB) *Config {
	ports, err := getAvailablePorts(2)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	return &Config{
		Dir:       dir,
		ClientURL: "http://" + net.JoinHostPort("localhost", strconv.Itoa(ports[0])),
		PeerURL:   "http://" + net.JoinHostPort("localhost", strconv.Itoa(ports[1])),
	}
}

var autoPortLock sync.Mutex

// RunEtcd starts an etcd server with a test configuration run through any
// provided tweak functions, and returns a client connected to it. The server
// is terminated when the test ends.
func RunEtcd(t testing.TB, tweakConfig ...func(cfg *Config)) *kubernetes.Client {
	t.Helper()
	bin := binaryPath(t)

	// lock until we successfully start the server on the ports we chose
	autoPortLock.Lock()
	defer autoPortLock.Unlock()
	var (
		cfg  *Config
		urls []string
		err  error
	)
	// reattempt up to three times if the port was taken
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cfg = newTestConfig(t)
		for _, f := range tweakConfig {
			f(cfg)
		}
		if cfg.ClientCertFile != "" {
			cfg.ClientURL = "https" + cfg.ClientURL[len("http"):]
		}
		var addrInUse bool
		urls, addrInUse, err = startServer(t, bin, cfg)
		if addrInUse && attempt < maxAttempts {
			t.Logf("error starting etcd, retrying: %v", err)
			continue
		}
		break
	}
	if err != nil {
		t.Fatal(err)
	}

	var tlsConfig *tls.Config
	if cfg.ClientCertFile != "" {
		info := transport.TLSInfo{CertFile: cfg.ClientCertFile, KeyFile: cfg.ClientKeyFile, TrustedCAFile: cfg.ClientTrustedCAFile}
		if tlsConfig, err = info.ClientConfig(); err != nil {
			t.Fatal(err)
		}
	}
	client, err := kubernetes.New(clientv3.Config{
		TLS:         tlsConfig,
		Endpoints:   urls,
		DialTimeout: 10 * time.Second,
		Logger:      zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel)).Named("etcd-client"),
	})
	if err != nil {
		t.Fatal(err)
	}
	kubernetesRecorder := storagetesting.NewKubernetesRecorder(client.Kubernetes)
	client.KV = storagetesting.NewKVRecorder(client.KV, kubernetesRecorder)
	client.Kubernetes = kubernetesRecorder
	return client
}

type lockedBuffer struct {
	sync.Mutex
	bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.String()
}

// startServer runs the helper and returns the client URLs once the server is
// ready. The second result says whether a listener port was taken.
func startServer(t testing.TB, bin string, cfg *Config) ([]string, bool, error) {
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, false, err
	}
	cmd := exec.Command(bin)
	stderr := &lockedBuffer{}
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, false, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	stop := func() {
		stdin.Close()
		select {
		case <-exited:
		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			<-exited
		}
	}

	if _, err := stdin.Write(append(configJSON, '\n')); err != nil {
		stop()
		return nil, false, err
	}
	lines := make(chan []byte, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadBytes('\n')
		lines <- line
		io.Copy(io.Discard, stdout)
	}()
	var line []byte
	select {
	case line = <-lines:
	case <-time.After(90 * time.Second):
		stop()
		return nil, false, fmt.Errorf("%s did not answer within 90s\n%s", binaryName, stderr)
	}
	var r reply
	if err := json.Unmarshal(line, &r); err != nil {
		stop()
		return nil, false, fmt.Errorf("%s: bad reply %q: %v\n%s", binaryName, line, err, stderr)
	}
	if r.Error != "" {
		stop()
		return nil, r.AddrInUse, errors.New(r.Error)
	}

	stopping := make(chan struct{})
	go func() {
		err := <-exited
		exited <- err
		select {
		case <-stopping:
		default:
			t.Errorf("%s exited: %v\n%s", binaryName, err, stderr)
		}
	}()
	t.Cleanup(func() {
		close(stopping)
		stop()
	})
	return r.ClientURLs, false, nil
}

// hostOf returns the host:port of a URL; tests use it to occupy a chosen port.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}
