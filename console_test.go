//go:build linux || zos || freebsd
// +build linux zos freebsd

/*
   Copyright The containerd Authors.

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

package console

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestWinSize(t *testing.T) {
	c, _, err := NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Resize(WinSize{
		Width:  11,
		Height: 10,
	}); err != nil {
		t.Error(err)
		return
	}
	size, err := c.Size()
	if err != nil {
		t.Error(err)
		return
	}
	if size.Width != 11 {
		t.Errorf("width should be 11 but received %d", size.Width)
	}
	if size.Height != 10 {
		t.Errorf("height should be 10 but received %d", size.Height)
	}
}

func testConsolePty(t *testing.T, newPty func() (Console, string, error)) {
	console, slavePath, err := newPty()
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()

	iteration := 10

	var (
		b  bytes.Buffer
		wg sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		io.Copy(&b, console)
		wg.Done()
	}()

	for i := 0; i < iteration; i++ {
		cmd := exec.Command("sh", "-c", "printf test")
		cmd.Stdin = slave
		cmd.Stdout = slave
		cmd.Stderr = slave

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	}
	slave.Close()
	wg.Wait()

	expectedOutput := ""
	for i := 0; i < iteration; i++ {
		expectedOutput += "test"
	}
	if out := b.String(); out != expectedOutput {
		t.Errorf("unexpected output %q", out)
	}
}

func TestConsolePty_NewPty(t *testing.T) {
	testConsolePty(t, NewPty)
}

func TestConsolePty_NewPtyFromFile(t *testing.T) {
	testConsolePty(t, func() (Console, string, error) {
		// Equivalent to NewPty().
		f, err := openpt()
		if err != nil {
			return nil, "", err
		}
		return NewPtyFromFile(f)
	})
}
