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

// Command etcd-testserver runs an embedded etcd for apiserver tests. It reads
// one JSON Config from stdin, prints the client URLs as JSON on stdout and
// exits when stdin closes, on SIGTERM, or when the server fails. It is a
// separate module so go.etcd.io/etcd/server/v3 stays out of the Kubernetes
// module graph; k8s.io/apiserver/pkg/storage/etcd3/testserver builds and runs it.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config is the subset of embed.Config the tests set. Keep in sync with
// k8s.io/apiserver/pkg/storage/etcd3/testserver.Config.
type Config struct {
	Dir                         string        `json:"dir"`
	ClientURL                   string        `json:"clientURL"`
	PeerURL                     string        `json:"peerURL"`
	QuotaBackendBytes           int64         `json:"quotaBackendBytes,omitempty"`
	WatchProgressNotifyInterval time.Duration `json:"watchProgressNotifyInterval,omitempty"`
	ClientCertFile              string        `json:"clientCertFile,omitempty"`
	ClientKeyFile               string        `json:"clientKeyFile,omitempty"`
	ClientTrustedCAFile         string        `json:"clientTrustedCAFile,omitempty"`
}

// Reply is the first line written to stdout.
type Reply struct {
	ClientURLs []string `json:"clientURLs,omitempty"`
	Error      string   `json:"error,omitempty"`
	AddrInUse  bool     `json:"addrInUse,omitempty"`
}

func start(c Config) (*embed.Etcd, error) {
	clientURL, err := url.Parse(c.ClientURL)
	if err != nil {
		return nil, err
	}
	peerURL, err := url.Parse(c.PeerURL)
	if err != nil {
		return nil, err
	}
	cfg := embed.NewConfig()
	cfg.Dir = c.Dir
	// Fine for a throwaway single member; makes the tests much faster.
	cfg.UnsafeNoFsync = true
	cfg.ListenPeerUrls = []url.URL{*peerURL}
	cfg.AdvertisePeerUrls = []url.URL{*peerURL}
	cfg.ListenClientUrls = []url.URL{*clientURL}
	cfg.AdvertiseClientUrls = []url.URL{*clientURL}
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	if c.QuotaBackendBytes != 0 {
		cfg.QuotaBackendBytes = c.QuotaBackendBytes
	}
	if c.WatchProgressNotifyInterval != 0 {
		cfg.WatchProgressNotifyInterval = c.WatchProgressNotifyInterval
	}
	if c.ClientCertFile != "" {
		cfg.ClientTLSInfo = transport.TLSInfo{CertFile: c.ClientCertFile, KeyFile: c.ClientKeyFile, TrustedCAFile: c.ClientTrustedCAFile}
	}
	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}
	cfg.ZapLoggerBuilder = embed.NewZapLoggerBuilder(logger.Named("etcd-server"))

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}
	select {
	case <-e.Server.ReadyNotify():
		return e, nil
	case <-time.After(60 * time.Second):
		e.Server.Stop()
		e.Close()
		return nil, errors.New("etcd took too long to start")
	}
}

func main() {
	var c Config
	if err := json.NewDecoder(os.Stdin).Decode(&c); err != nil {
		fmt.Fprintf(os.Stderr, "etcd-testserver: reading config: %v\n", err)
		os.Exit(2)
	}
	enc := json.NewEncoder(os.Stdout)
	e, err := start(c)
	if err != nil {
		_ = enc.Encode(Reply{Error: err.Error(), AddrInUse: errors.Is(err, syscall.EADDRINUSE)})
		os.Exit(1)
	}
	_ = enc.Encode(Reply{ClientURLs: e.Server.Cluster().ClientURLs()})

	stdinClosed := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		close(stdinClosed)
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stdinClosed:
	case <-sigs:
	case err := <-e.Err():
		fmt.Fprintf(os.Stderr, "etcd-testserver: %v\n", err)
		e.Close()
		os.Exit(1)
	}
	e.Close()
}
