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

// This file contains synchronized (thread-safe) versions of Fortuna and its
// Generator.

package fortuna

import (
	"os"
	"sync"
)

// SynchronizedPRNG is a thread-safe PRNG which can be used with multiple
// goroutines.
type SynchronizedPRNG struct {
	lock sync.Mutex
	prng PRNG
}

// we don't want everybody to call this
func newSynchronizedFortuna(f *Fortuna) *SynchronizedPRNG {
	result := new(SynchronizedPRNG)
	//	result.prng = newFortuna()
	result.prng = f
	return result
}

// it's ok for this to be public
func NewSynchronizedGenerator() *SynchronizedPRNG {
	result := new(SynchronizedPRNG)
	result.prng = NewGenerator()
	return result
}

func (sg *SynchronizedPRNG) Seed(seed int64) {
	sg.lock.Lock()
	sg.prng.Seed(seed)
	sg.lock.Unlock()
}

func (sg *SynchronizedPRNG) Int63() int64 { return int63(sg) }

func (sg *SynchronizedPRNG) Read(buffer []byte) (n int, err os.Error) {
	sg.lock.Lock()
	defer sg.lock.Unlock()
	return sg.prng.Read(buffer)
}

// Compile-time assertion
var _ PRNG = (*SynchronizedPRNG)(nil)
