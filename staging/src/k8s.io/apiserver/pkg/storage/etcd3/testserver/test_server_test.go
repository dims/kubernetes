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
	"context"
	"net"
	"testing"
	"time"
)

// TestRunEtcdRecoversFromTakenPort claims a port that was selected for etcd
// before it gets a chance to bind, reproducing what happens when another
// process takes the port first, and verifies that RunEtcd recovers instead of
// failing the test. Both listeners are covered because etcd binds the peer
// listener before the client one, so either can be the one that loses the race.
func TestRunEtcdRecoversFromTakenPort(t *testing.T) {
	for _, tc := range []struct {
		name string
		addr func(cfg *Config) string
	}{
		{name: "client", addr: func(cfg *Config) string { return hostOf(cfg.ClientURL) }},
		{name: "peer", addr: func(cfg *Config) string { return hostOf(cfg.PeerURL) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := int64(0)
			client := RunEtcd(t, func(cfg *Config) {
				if called == 0 {
					// grab the allocated port on the first attempt
					taken := tc.addr(cfg)
					l, err := net.Listen("tcp", taken)
					if err != nil {
						t.Fatalf("failed to claim %s: %v", taken, err)
					}
					t.Cleanup(func() { _ = l.Close() })
				}
				called++
			})

			if called < 2 {
				t.Fatalf("expected config to be constructed at least twice, got %d", called)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if _, err := client.KV.Get(ctx, "/registry"); err != nil {
				t.Fatalf("etcd did not serve requests after moving to a new port: %v", err)
			}
		})
	}
}
