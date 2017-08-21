package ipset

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Handle struct {
	cmd               *exec.Cmd
	args              []string
	ipsetBin          string
	stdin             io.WriteCloser
	stdout            io.ReadCloser
	stderr            io.ReadCloser
	isRunning         func(*Handle) bool
	handleInteractive bool
}

var (
	// Default strategy is to allow os figure out where binary is.
	defaultIpsetBin = "ipset"

	// Default strategy is to silence messages that exist already and
	// use interactive mode for Write() call.
	defaultIpsetArgs = []string{"--exist", "-"}
)

func defaultStateFunc(h *Handle) bool {
	if h == nil || h.cmd == nil {
		return false
	}

	started := h.cmd.Process != nil
	exited := h.cmd.ProcessState != nil

	return started && !exited
}

// New takes a variable amount of option functions and returns configured *Handle.
func New(options ...OptFunc) (*Handle, error) {
	var err error
	h := Handle{
		ipsetBin:  defaultIpsetBin,
		isRunning: defaultStateFunc,

		// debug
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

func (h *Handle) Start() error {
	return h.cmd.Start()
}

func (h *Handle) Wait(ctx context.Context) error {
	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

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

func (h *Handle) Write(p []byte) (int, error) {
	if !h.isRunning(h) {
		return 0, errors.New("Process not started ")
	}

	return h.stdin.Write(p)
}

func (h *Handle) Quit() error {
	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, "quit\n")
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil

}

func (h *Handle) Add(s *Set) error {
	if s == nil {
		return nil
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderAdd))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

func (h *Handle) Swap(s1, s2 *Set) error {
	if s1 == nil || s2 == nil {
		return errors.New("swap() does not accept nil")
	}

	if s1.Type != s2.Type {
		return errors.New(fmt.Sprintf("can not swap sets %s.%s <-> %s.%s due to different types", s1, s1.Type, s2, s2.Type))
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, fmt.Sprintf("swap %s %s\n", s1.Name, s2.Name))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

func (h *Handle) Delete(s *Set) error {
	if s == nil {
		return nil
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderDelete))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

func (h *Handle) Create(s *Set) error {
	if s == nil {
		return nil
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderCreate))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

func (h *Handle) Flush(s *Set) error {
	if s == nil {
		return nil
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderFlush))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

func (h *Handle) Destroy(s *Set) error {
	if s == nil {
		return nil
	}

	if !h.isRunning(h) {
		return errors.New("Process not started ")
	}

	_, err := io.WriteString(h, s.Render(RenderDestroy))
	if err != nil {
		return errors.Wrap(err, "failed to write set")
	}

	return nil
}

// TODO return ok is version is compatible and version string.
func IpsetVersion(options ...OptFunc) (bool, string) {
	return true, ""
}

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

	handle, err := New(options...)
	if err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG in oneshot, executing %s %s\n", handle.ipsetBin, handle.args)

	return handle.cmd.CombinedOutput()
}

func Add(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderAdd, options...)
}

func Create(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderCreate, options...)
}

func Delete(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderDelete, options...)
}

func Destroy(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderDestroy, options...)
}

func Flush(set *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set, nil, RenderFlush, options...)
}

func Swap(set1, set2 *Set, options ...OptFunc) ([]byte, error) {
	if set1 == nil || set2 == nil {
		return nil, errors.New("must have exactly 2 non nil sets for swap")
	}
	return oneshot(set1, set2, RenderSwap, options...)
}

func Rename(set1, set2 *Set, options ...OptFunc) ([]byte, error) {
	if set1 == nil || set2 == nil {
		return nil, errors.New("must have exactly 2 non nil sets for rename")
	}
	return oneshot(set1, set2, RenderRename, options...)
}

func Test(set1 *Set, options ...OptFunc) ([]byte, error) {
	return oneshot(set1, nil, RenderTest, options...)
}

func (h *Handle) IsSuccessful() bool {
	if h.cmd == nil || h.cmd.ProcessState == nil {
		return false
	}

	return h.cmd.ProcessState.Success()
}

// OptFunc is a signature for option functions for use with New()
// TODO Rename into HandleOptFunc
type OptFunc func(*Handle) error

// HandleWithBin is an options for New() to use non default location of ipset binary.
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

// HandleWithArgs is an options for New() to use non default arguments for call to ipset binary.
// TODO rename HandleArgs
func HandleWithArgs(args ...string) OptFunc {
	return func(h *Handle) error {
		h.args = args
		return nil
	}
}

// HandleAppendArgs is an options for New() to use non default arguments for call to ipset binary.
// TODO rename HandleArgs
func HandleAppendArgs(args ...string) OptFunc {
	return func(h *Handle) error {
		h.args = append(h.args, args...)
		return nil
	}
}

// Load ipset config from system.
func Load(options ...OptFunc) (*Ipset, error) {
	options = append(options, HandleWithArgs("save", "-o", "xml"))
	handle, err := New(options...)
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

//TODO AddAll/DeleteAll ...

/*
// Restore given ipsets.

func Restore(cmd exec.Cmd, ipsets Ipsets, options []string) error {
	// open stdin to `ipset -` and Save() into it.
	return nil
}

/*
// Save renders given ipsets into ipset-save format.
func Save(ipsets Ipsets) string {
	return ""
}

// Destroy all sets mentioned in given ipsets. Destroys everything if no Sets.
func Destroy(cmd exec.Cmd, ipsets Ipsets) error {
	return nil
}

// Delete all elements mentioned in ipsets.
func Delete(cmd exec.Cmd, ipsets Ipsets, options []string) error {
	return nil
}
*/

func renderSet2args(iset *Ipset, rType RenderType) []string {
	return strings.Split(strings.TrimSpace(iset.Render(rType)), " ")
}
