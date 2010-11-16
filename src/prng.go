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

// This package implements the Fortuna strong pseudo random number generator.
// Fortuna is designed by Niels Ferguson and Bruce Schneier and is described
// in "Practical Cryptography" (ISBN 0-471-22357-3) pp. 161..184.
package fortuna

import (
	sha "crypto/sha256"
	"hash"
	"io"
	"once"
	"os"
	"rand"
	"strconv"
	"sync"
)

// PRNG is an interface to an unending stream of pseudo-random data.
type PRNG interface {
	// Source represents a source of uniformly-distributed pseudo-random int64
	// values in the range [0, 1<<63).
	//
	// It consists of two methods:
	//
	// Seed(seed int64) // to seed the generator, and
	// Int63() int64 - to return a 63-bit integer.
	rand.Source

	// Reader is the interface that wraps the basic Read method:
	// Read(buffer []byte) (n int, err os.Error)
	//
	// Read reads up to len(buffer) bytes into buffer.  It returns the number
	// of bytes actually read which in the case of PRNGs will always be
	// len(buffer).
	//
	// There are three noticeable differences in the case of PRNGs with regard
	// to the conventional Read contract:
	//
	// 1. Read calls will block until enough data is available to fill buffer,
	// 2. Read will always return len(buffer) and nil, unless...
	// 3. Read will return -1 and an error iff the PRNG is not yet seeded.
	io.Reader
}

// Fortuna is the main type which implements PRNG
type Fortuna struct {
	generator      *Generator  // the generator which also implements PRNG
	timeLastSeeded int64       // time in milliseconds of last [re-]seeding
	timesSeeded    uint64      // how many times this Fortuna was seeded so far
	lock           sync.Mutex  // guard against concurrent access to pools
	pools          []hash.Hash // 32 running SHA-256 message digests
	pool0Count     int         // number of bytes added to pool #0 since last reseed
}

var globalFortuna = newFortuna()
var globalSyncronizedFortuna *SynchronizedPRNG

func init() {
	globalSyncronizedFortuna = newSynchronizedFortuna(globalFortuna)
	go updateSeedPeriodically(globalFortuna)
	logInfo("init(): Fortuna is ready...")
	once.Do(startAccumulator)
}

func newFortuna() *Fortuna {
	result := new(Fortuna)
	result.generator = NewUnseededGenerator()
	result.pools = make([]hash.Hash, 32)
	for i := range result.pools {
		result.pools[i] = sha.New()
	}
	return result
}

// GetFortuna returns the synchronized Fortuna singleton.
func GetFortuna() *SynchronizedPRNG { return globalSyncronizedFortuna }

// SHA-256 work block size in bytes
var minPoolSize = 64

// Seed uses the provided seed value to initialize the generator to a
// deterministic state.
// It is also the implementation of the 1st method in rand.Source interface.
func (f *Fortuna) Seed(seed int64) { globalUserSeed.setSeed(seed) }

// Int63 returns a non-negative pseudo-random 63-bit long integer as an int64.
// It is also the implementation of the 2nd method in rand.Source interface.
func (f *Fortuna) Int63() int64 { return int63(f) }

// Read reads up to len(buffer) bytes into buffer.
func (f *Fortuna) Read(buffer []byte) (n int, err os.Error) {
	return f.generator.Read(buffer)
}

func (f *Fortuna) updatePool(poolNdx, id int, entropy []byte) {
	logTrace("--> updatePool(" + strconv.Itoa(poolNdx) + ", " + strconv.Itoa(id) + ", [" + strconv.Itoa(len(entropy)) + "]byte)")
	f.lock.Lock()
	// NOTE (rsn) - no entropy is expected to be > 32 bytes (p. 176)
	f.pools[poolNdx].Write([]byte{byte(id), byte(len(entropy))})
	f.pools[poolNdx].Write(entropy)
	f.lock.Unlock()
	if poolNdx == 0 {
		f.pool0Count += len(entropy)
		if f.pool0Count > minPoolSize && currentTimeMillis()-f.timeLastSeeded > 100 {
			f.reseed()
		}
	}
	logTrace("<-- updatePool()")
}

func (f *Fortuna) reseed() {
	logTrace("--> F.reseed()")
	// on the n-th reseeding, pool k is used only if 2**k divides n
	f.timesSeeded++
	logDebug("F.reseed(): timesSeeded = " + strconv.Uitoa64(f.timesSeeded))
	f.lock.Lock()
	for k := 0; k < 32; k++ {
		if f.timesSeeded%(1<<uint64(k)) == 0 {
			poolBytes := f.pools[k].Sum()
			f.generator.md.Write(poolBytes)
		}
	}
	f.lock.Unlock()
	f.generator.reseed()
	f.pool0Count = 0
	f.timeLastSeeded = currentTimeMillis()
	if f.timesSeeded == 1 {
		logInfo("F.reseed(): Fortuna is ready and able...")
	}
	// IMPLEMENTATION NOTE (rsn) - we're supposed to save the random seed
	// just before a shutdown.  since we cannot do that in Go we opt for
	// saving it every 10 minutes and every 1000 reseedings.
	if f.timesSeeded%timesSeededPeriod == 0 {
		go updateSeedFile(f)
	}
	logTrace("<-- F.reseed()")
}

// Compile-time assertion that Fortuna implements PRNG
var _ PRNG = (*Fortuna)(nil)
