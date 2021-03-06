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

// This file contains the Fortuna Accumulator and related Entropy Sources
// structures and functions.

package fortuna

import (
	"io"
	"os"
	"sync"
	"time"
)

type entropySource interface {
	// Returns a unique ID in the range [0..255] identifying this source.
	getID() int

	// Returns the index of the Fortuna Pool to receive the next entropy bytes.
	getPoolNdx() int

	// Returns available entropy bytes and potential OS error.
	getEntropy() ([]byte, os.Error)
}

var globalDevRandom = &devSource{id: 0, name: "/dev/random"}
var globalUserSeed = new(userSeed)
var globalDevURandom = &devSource{id: 2, name: "/dev/urandom"}
// TODO (rsn) - implement more collectors; e.g. a GPG seed reader

// Starts the known entropy collectors each in its own goroutine.
func startAccumulator() {
	go start(globalDevRandom)
	go start(globalUserSeed)
	go start(globalDevURandom)
}

func start(es entropySource) {
	logInfo("start(): Starting collector loop of entropy " + toString(es))
	for true {
		logDebug("start(): Getting entropy from " + toString(es))
		entropy, err := es.getEntropy() // will block until enough entropy is ready
		logDebug("Got entropy from " + toString(es))
		if err != nil {
			logWarn("start(): Unexpected error getting entropy from " + toString(es) + ": " + err.String())
			// TODO (rsn) - investigate whether we should retry a few times
			// and if error persists, bail out altogether
		} else { // update Fortuna's pool at index poolNdx
			logDebug("start(): Updating Fortuna w/ entropy from " + toString(es))
			globalFortuna.updatePool(es.getPoolNdx(), es.getID(), entropy)
		}
	}
}

type devSource struct {
	lock    sync.Mutex
	id      int
	name    string
	f       *os.File
	poolNdx int
}

func (r *devSource) getID() int { return r.id }

func (r *devSource) getPoolNdx() int {
	result := r.poolNdx
	r.poolNdx = (r.poolNdx + 1) % 32
	return result
}

func (r *devSource) getEntropy() ([]byte, os.Error) {
	logTrace("--> " + r.name + ".getEntropy()")
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.f == nil {
		logInfo(r.name + ".getEntropy(): About to open device")
		f, err := os.OpenFile(r.name, os.O_RDONLY, 0)
		if f == nil {
			logError("Unable to open " + r.name + ": " + err.String())
			return nil, err
		}
		r.f = f
	}
	buffer := make([]byte, 32) // maximum event size (p. 176)
	logDebug(r.name + ".getEntropy(): About to read 32 bytes")
	n, err := io.ReadFull(r.f, buffer)
	logTrace("<-- " + r.name + ".getEntropy()")
	return buffer[0:n], err
}

type userSeed struct {
	lock    sync.Mutex
	seed    int64
	ready   bool
	poolNdx int
}

func (r *userSeed) getID() int { return 1 }

func (r *userSeed) getPoolNdx() int {
	result := r.poolNdx
	r.poolNdx = (r.poolNdx + 1) % 32
	return result
}

func (r *userSeed) getEntropy() ([]byte, os.Error) {
	logTrace("--> userSeed.getEntropy()")
	for true {
		r.lock.Lock()
		if r.ready {
			break
		}
		r.lock.Unlock()
		time.Sleep(1e9) // 1 second
	}
	result := toBytes(r.seed)
	r.ready = false
	r.lock.Unlock()
	logTrace("<-- userSeed.getEntropy()")
	return result, nil
}

func (r *userSeed) setSeed(seed int64) {
	r.lock.Lock()
	r.seed = seed
	r.ready = true
	r.lock.Unlock()
}

// Compile-time assertions
var _ entropySource = (*devSource)(nil)
var _ entropySource = (*userSeed)(nil)
