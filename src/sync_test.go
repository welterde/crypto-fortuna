// Copyright 2010 Type & Graphics Pty. Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fortuna

import (
	"testing"
)

// TestSynchronizedGenerator tests that a syncronized Fortuna PRNG does not
// generate random series which overlap when used in 2 goroutines.
func TestSynchronizedGenerator(t *testing.T) {
	prng := NewSynchronizedGenerator()
	testSyncronization(t, prng)
}

func testSyncronization(t *testing.T, prng PRNG) {
	x := make([]int64, 5000)
	y := make([]int64, 5000)
	c1 := make(chan bool)
	c2 := make(chan bool)
	go newRandoms(prng, x, c1)
	go newRandoms(prng, y, c2)
	<-c1
	<-c2
	for i, xi := range x {
		for j, yj := range y {
			if xi == yj {
				t.Errorf("PRNG.Int63, found an overlap x[#%d], y[%d]: %d, %d", i, j, xi, yj)
				break
			}
		}
	}
}

func newRandoms(prng PRNG, x []int64, ch chan<- bool) {
	for i := range x {
		x[i] = prng.Int63()
	}
	ch <- true
}
