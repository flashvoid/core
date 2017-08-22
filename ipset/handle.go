// Copyright (c) 2017 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package ipset

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Handle for interactive ipset session,
// it keeps state and allocated resources together.
// Handle is open when and underlying process
// initizlized and started and hasn't exited yet.
type Handle struct {
	// arguments used to initialize underlying command.
	args []string
	// binary used to initialize underlying command.
	ipsetBin string

	// underlying command,
	cmd *exec.Cmd

	// handleInteractive flag tells to initialize
	// handle for interactive usage.
	handleInteractive bool
	stdin             io.WriteCloser
	stdout            io.ReadCloser
	stderr            io.ReadCloser

	// isOpen indicates if handle is open.
	isOpen func(*Handle) bool
}

var (
	// Default strategy is to allow os figure out where binary is.
	defaultIpsetBin = "ipset"

	// Default strategy is to silence messages that exist already and
	// use interactive mode for Write() call.
	defaultIpsetArgs = []string{"--exist", "-"}
)

// NewHandle takes a variable amount of option functions and returns configured *Handle.
func NewHandle(options ...OptFunc) (*Handle, error) {
	var err error
	h := Handle{
		ipsetBin:          defaultIpsetBin,
		isOpen:            defaultStateFunc,
		handleInteractive: true,
	}

	for _, opt := range options {
		err = opt(&h)
		if err != nil {
			return nil, err
		}

	}

	if len(h.args) == 0 {
		h.args = defaultIpsetArgs
	}

	h.cmd = exec.Command(h.ipsetBin, h.args...)

	if h.handleInteractive {
		h.stdin, err = h.cmd.StdinPipe()
		h.stderr, err = h.cmd.StderrPipe()
		h.stdout, err = h.cmd.StdoutPipe()
	}

	return &h, err
}

// Start interactive session, normally transfers the Handle
// into the open state.
func (h *Handle) Start() error {
	return h.cmd.Start()
}

// Wait for handle to stop interactive session and deallocate resources.
func (h *Handle) Wait(ctx context.Context) error {
	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	// users should call Quit() before calling Wait()
	// but just in case they don't.
	_ = h.Quit()

	success := make(chan struct{})
	go func() {
		h.cmd.Wait()
		close(success)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-success:
			return nil
		}
	}

	return nil
}

// Write into the open handle.
func (h *Handle) Write(p []byte) (int, error) {
	if !h.isOpen(h) {
		return 0, errors.New("Process not started ")
	}

	return h.stdin.Write(p)
}

// Read from the open handle.
func (h *Handle) Read(p []byte) (int, error) {
	if !h.isOpen(h) {
		return 0, errors.New("Process not started ")
	}

	return h.stdout.Read(p)
}

// StdErr provides access to stderr of running process.
func (h *Handle) StdErr() (io.Reader, error) {
	if !h.isOpen(h) {
		return nil, errors.New("Process not started ")
	}

	return h.stderr, nil
}

// Quit interactive session.
func (h *Handle) Quit() error {
	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, "quit\n")
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil

}

// Add members of sets to ipset through the open handle.
func (h *Handle) Add(s renderer) error {
	if s == nil {
		return nil
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderAdd))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Swap contents of 2 sets through the open handle.
func (h *Handle) Swap(s1, s2 *Set) error {
	if s1 == nil || s2 == nil {
		return errors.New("swap() does not accept nil")
	}

	if s1.Type != s2.Type {
		return errors.New(fmt.Sprintf("can not swap sets %s.%s <-> %s.%s due to different types", s1, s1.Type, s2, s2.Type))
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, fmt.Sprintf("swap %s %s\n", s1.Name, s2.Name))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Delete memebers of sets from ipset through the open handle.
func (h *Handle) Delete(s renderer) error {
	if s == nil {
		return nil
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderDelete))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Create sets and members in ipset through the open handle.
func (h *Handle) Create(s renderer) error {
	if s == nil {
		return nil
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderCreate))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Flush sets in ipset through the open handle.
// Warning. Will flush all sets if no sets are given.
func (h *Handle) Flush(s renderer) error {
	if s == nil {
		return nil
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderFlush))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Destroy sets in ipset through the open handle.
// Warning. Will destroy everything in ipset if no sets are given.
func (h *Handle) Destroy(s renderer) error {
	if s == nil {
		return nil
	}

	if !h.isOpen(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderDestroy))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// Add members of sets to ipset.
func Add(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderAdd, options...)
}

// Create sets and members in ipset.
func Create(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderCreate, options...)
}

// Delete memebers from ipset.
func Delete(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderDelete, options...)
}

// Destroy sets in ipset. Destroys everything if no sets are given.
func Destroy(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderDestroy, options...)
}

// Flush sets in ipset. Flushes everythin if no sets are given.
func Flush(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderFlush, options...)
}

// Swap contents of 2 sets.
func Swap(set1, set2 *Set, options ...OptFunc) ([]byte, error) {
	if set1 == nil || set2 == nil {
		return nil, errors.New("must have exactly 2 non nil sets for swap")
	}
	return oneshot(set1, set2, RenderSwap, options...)
}

// Rename set.
func Rename(set1, set2 *Set, options ...OptFunc) ([]byte, error) {
	if set1 == nil || set2 == nil {
		return nil, errors.New("must have exactly 2 non nil sets for rename")
	}
	return oneshot(set1, set2, RenderRename, options...)
}

// Version captures version of ipset and parses it for later verification.
func Version(options ...OptFunc) (*IpsetVersion, error) {
	options = append(options, HandleWithArgs("version"), handleNonInteractive())

	handle, err := NewHandle(options...)
	if err != nil {
		return nil, err
	}

	data, err := handle.cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	sdata := strings.TrimSpace(string(data))
	sdata = strings.Replace(sdata, "v", "", -1)
	sdata = strings.Replace(sdata, ",", "", -1)
	temp := strings.Split(sdata, " ")
	if len(temp) < 2 {
		return nil, errors.Errorf("can not parse %s as a version string", string(data))
	}

	verUtil, err := strconv.ParseFloat(temp[1], 0)
	if err != nil {
		return nil, errors.Wrapf(err, "can not parse %s as a version string", string(data))
	}

	verProto, err := strconv.Atoi(temp[len(temp)-1])
	if err != nil {
		return nil, errors.Wrapf(err, "can not parse %s as a version string", string(data))
	}

	return &IpsetVersion{Util: verUtil, Proto: verProto}, nil
}

const (
	SupportedVersionUtil float64 = 6.29
	SupportVersionProto          = 6
)

// Version prepresents ipset version.
type IpsetVersion struct {
	Util  float64
	Proto int
}

// Check that given version is supported.
func (v *IpsetVersion) Check() bool {
	if v == nil {
		return false
	}
	return v.Util >= SupportedVersionUtil && v.Proto >= SupportVersionProto
}

// Tests existanse of a set or existance of memeber in a set.
func Test(set1 *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set1, nil, RenderTest, options...)
}

// oneshot creates a temporary handle to execute one command.
func oneshot(set1, set2 *Set, rType RenderType, options ...OptFunc) ([]byte, error) {
	iset := &Ipset{}
	if set1 != nil {
		iset.Sets = append(iset.Sets, set1)
	}
	if set2 != nil {
		iset.Sets = append(iset.Sets, set2)
	}

	args := renderSet2args(iset, rType)
	options = append(options, HandleAppendArgs(args...), handleNonInteractive())

	handle, err := NewHandle(options...)
	if err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG in oneshot, executing %s %s\n", handle.ipsetBin, handle.args)

	return handle.cmd.CombinedOutput()
}

func (h *Handle) IsSuccessful() bool {
	if h.cmd == nil || h.cmd.ProcessState == nil {
		return false
	}

	return h.cmd.ProcessState.Success()
}

// OptFunc is a signature for option functions that change
// configuration of handle.
type OptFunc func(*Handle) error

// HandleWithBin is an options to use non default location of ipset binary.
func HandleWithBin(bin string) OptFunc {
	return func(h *Handle) error {
		h.ipsetBin = bin
		return nil
	}
}

// handleNonInteractive configures handle for non-interactive mode.
func handleNonInteractive() OptFunc {
	return func(h *Handle) error {
		h.handleInteractive = false
		return nil
	}
}

// HandleWithArgs is an options for to use non default arguments for call to ipset binary.
func HandleWithArgs(args ...string) OptFunc {
	return func(h *Handle) error {
		h.args = args
		return nil
	}
}

// HandleAppendArgs is an options that adds more args after HandleWithArgs was used.
func HandleAppendArgs(args ...string) OptFunc {
	return func(h *Handle) error {
		h.args = append(h.args, args...)
		return nil
	}
}

// Load ipset config from system.
func Load(options ...OptFunc) (*Ipset, error) {
	options = append(options, HandleWithArgs("save", "-o", "xml"))
	handle, err := NewHandle(options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ipset handler")
	}

	err = handle.Start()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start ipset")
	}

	var ipset Ipset

	err = xml.NewDecoder(handle.stdout).Decode(&ipset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ipset config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	handle.Wait(ctx)

	return &ipset, nil

}

// LoadFromFile loads ipset config form xml file produced with ipset save -o xml.
func LoadFromFile(filename string) (*Ipset, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ipset config from file")
	}

	var ipset Ipset
	err = xml.Unmarshal(data, &ipset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ipset config from file")
	}

	return &ipset, nil
}

// renderSet2args is a helper that renders sets for use with oneshot functions.
func renderSet2args(iset renderer, rType RenderType) []string {
	// by deafult Render produces interactive version
	// of commands that have \n at the end,
	// this removes it for compatibility with
	// oneshot version of functions.
	trimmed := strings.TrimSpace(iset.Render(rType))
	return strings.Split(trimmed, " ")
}

// renderer abstracts render behavior which allows treating
// Ipset and Set objects similarly.
type renderer interface {
	Render(rType RenderType) string
}

// defaultStateFunc is true when underlying process started
// and not exited.
func defaultStateFunc(h *Handle) bool {
	if h == nil || h.cmd == nil {
		return false
	}

	started := h.cmd.Process != nil
	exited := h.cmd.ProcessState != nil

	return started && !exited
}
