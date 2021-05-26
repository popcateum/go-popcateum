// Copyright 2016 The go-popcateum Authors
// This file is part of go-popcateum.
//
// go-popcateum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-popcateum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-popcateum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/popcateum/go-popcateum/internal/cmdtest"
	"github.com/popcateum/go-popcateum/rpc"
)

func tmpdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "gpop-test")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

type testgpop struct {
	*cmdtest.TestCmd

	// template variables for expect
	Datadir   string
	Popcatbase string
}

func init() {
	// Run the app if we've been exec'd as "gpop-test" in runGpop.
	reexec.Register("gpop-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func TestMain(m *testing.M) {
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

// spawns gpop with the given command line args. If the args don't set --datadir, the
// child g gets a temporary data directory.
func runGpop(t *testing.T, args ...string) *testgpop {
	tt := &testgpop{}
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, arg := range args {
		switch arg {
		case "--datadir":
			if i < len(args)-1 {
				tt.Datadir = args[i+1]
			}
		case "--miner.popcatbase":
			if i < len(args)-1 {
				tt.Popcatbase = args[i+1]
			}
		}
	}
	if tt.Datadir == "" {
		tt.Datadir = tmpdir(t)
		tt.Cleanup = func() { os.RemoveAll(tt.Datadir) }
		args = append([]string{"--datadir", tt.Datadir}, args...)
		// Remove the temporary datadir if something fails below.
		defer func() {
			if t.Failed() {
				tt.Cleanup()
			}
		}()
	}

	// Boot "gpop". This actually runs the test binary but the TestMain
	// function will prevent any tests from running.
	tt.Run("gpop-test", args...)

	return tt
}

// waitForEndpoint attempts to connect to an RPC endpoint until it succeeds.
func waitForEndpoint(t *testing.T, endpoint string, timeout time.Duration) {
	probe := func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		c, err := rpc.DialContext(ctx, endpoint)
		if c != nil {
			_, err = c.SupportedModules()
			c.Close()
		}
		return err == nil
	}

	start := time.Now()
	for {
		if probe() {
			return
		}
		if time.Since(start) > timeout {
			t.Fatal("endpoint", endpoint, "did not open within", timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}