/*
Copyright 2021 The Kubernetes Authors.

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

package eventclock

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

func TestRealEventClock(t *testing.T) {
	ec := Real{}
	var numDone int32
	now := ec.Now()
	const batchSize = 100
	times := make(chan time.Time, batchSize+1)
	try := func(abs bool, d time.Duration) {
		f := func(u time.Time) {
			realD := ec.Since(now)
			atomic.AddInt32(&numDone, 1)
			times <- u
			if realD < d {
				t.Errorf("Asked for %v, got %v", d, realD)
			}
		}
		if abs {
			ec.EventAfterTime(f, now.Add(d))
		} else {
			ec.EventAfterDuration(f, d)
		}
	}
	try(true, time.Millisecond*3300)
	for i := 0; i < batchSize; i++ {
		d := time.Duration(rand.Intn(30)-3) * time.Millisecond * 100
		try(i%2 == 0, d)
	}
	// The latest event is scheduled for 3.3s. Poll with a generous deadline
	// instead of a fixed sleep so the test tolerates scheduling jitter on
	// loaded hosts (e.g. ppc64le CI runners).
	deadline := time.Now().Add(15 * time.Second)
	for atomic.LoadInt32(&numDone) != batchSize+1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&numDone) != batchSize+1 {
		t.Errorf("Got only %v events", atomic.LoadInt32(&numDone))
	}
	lastTime := now
	for i := 0; i <= batchSize; i++ {
		nextTime := <-times
		if nextTime.Before(now) {
			continue
		}
		dt := nextTime.Sub(lastTime) / (50 * time.Millisecond)
		if dt < 0 {
			t.Errorf("Got %s after %s", nextTime, lastTime)
		}
		lastTime = nextTime
	}
}
