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
	"time"
)

// TestFortuna_1 tests that 2 consecutive random numbers are not equal.
func TestFortuna_1(t *testing.T) {
	prng := GetFortuna()
	seed(prng)
	v_ := prng.Int63()
	for i := 0; i < 100000; i++ {
		v := prng.Int63()
		if v == v_ {
			t.Errorf("Fortuna.Int63, 2 consecutive 63-bit values are equal at iter #%d: %d", i, v)
			break
		}
	}
}

// TestFortuna_N tests that a randomly generated number was not among the last
// generated 100 ones.
func TestFortuna_N(t *testing.T) {
	prng := GetFortuna()
	seed(prng)
	// generate the first 100 randoms
	ring := make([]int64, 100)
	for i := 0; i < 100; i++ {
		ring[i] = prng.Int63()
	}
	n := 0
	for i := 100; i < 100000; i++ {
		v := prng.Int63()
		for i := 0; i < 100; i++ {
			if v == ring[i] {
				t.Errorf("Fortuna.Int63, a 63-bit value is equal to one already generated previosuly at iter #%d: %d", i, v)
				break
			}
		}
		n++
		n %= 100
		ring[n] = v
	}
}

func seed(prng *SynchronizedPRNG) {
	for i := 0; i < 8*32; i++ {
		prng.Seed(time.Nanoseconds())
		time.Sleep(1e6) // 1 ms
	}
}
