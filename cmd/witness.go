// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	f_note "github.com/transparency-dev/formats/note"
	"github.com/transparency-dev/witness/monitoring"
	"github.com/transparency-dev/witness/monitoring/prometheus"
	"github.com/transparency-dev/witness/omniwitness"

	"golang.org/x/mod/sumdb/note"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usbarmory/tamago-example/shell"
)

const (
	witnessName = "tamago-example-ephemeral-witness"
	witnessPort = 8080
)

func init() {
	shell.Add(shell.Cmd{
		Name: "witness",
		Help: "start/inspect transparency.dev omniwitness",
		Fn:   witnessCmd,
	})
}

// NewPersistence returns a persistence object that lives only in memory.
func NewPersistence() omniwitness.LogStatePersistence {
	return &inMemoryPersistence{
		checkpoints: make(map[string]checkpointState),
	}
}

type checkpointState struct {
	rawChkpt     []byte
	compactRange []byte
}

type inMemoryPersistence struct {
	// mu allows checkpoints to be read concurrently, but
	// exclusively locked for writing.
	mu          sync.RWMutex
	checkpoints map[string]checkpointState
}

var witnessLogs omniwitness.LogStatePersistence

func (p *inMemoryPersistence) Init() error {
	return nil
}

func (p *inMemoryPersistence) Logs() (res []string, _ error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	res = make([]string, 0, len(p.checkpoints))

	for k := range p.checkpoints {
		res = append(res, k)
	}

	return
}

func (p *inMemoryPersistence) ReadOps(logID string) (omniwitness.LogStateReadOps, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var cp *checkpointState

	if got, ok := p.checkpoints[logID]; ok {
		cp = &got
	}

	return &readWriter{
		read: cp,
	}, nil
}

func (p *inMemoryPersistence) WriteOps(logID string) (omniwitness.LogStateWriteOps, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var cp *checkpointState

	if got, ok := p.checkpoints[logID]; ok {
		cp = &got
	}

	return &readWriter{
		write: func(old *checkpointState, new checkpointState) error {
			return p.expectAndWrite(logID, old, new)
		},
		read: cp,
	}, nil
}

func (p *inMemoryPersistence) expectAndWrite(logID string, old *checkpointState, new checkpointState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	got, found := p.checkpoints[logID]

	if old != nil {
		if !found {
			return fmt.Errorf("expected old state %v but no state found when updating log %s", *old, logID)
		}
		if !reflect.DeepEqual(*old, got) {
			return fmt.Errorf("expected old state %v but got %s when updating log %s", *old, got, logID)
		}
	} else {
		if found {
			return fmt.Errorf("expected no state but found %v when updating log %s", got, logID)
		}
	}

	p.checkpoints[logID] = new

	return nil
}

type readWriter struct {
	write         func(*checkpointState, checkpointState) error
	read, toStore *checkpointState
}

func (rw *readWriter) GetLatest() ([]byte, error) {
	if rw.read == nil {
		return nil, status.Error(codes.NotFound, "no checkpoint found")
	}

	return rw.read.rawChkpt, nil
}

func (rw *readWriter) Set(c []byte) error {
	rw.toStore = &checkpointState{
		rawChkpt: c,
	}

	return rw.write(rw.read, *rw.toStore)
}

func (rw *readWriter) Close() error {
	return nil
}

func dumpWitnessLogs() (s string, err error) {
	logs, err := witnessLogs.Logs()

	if err != nil {
		return "", fmt.Errorf("failed to get log list, %v", err)
	}

	for _, logID := range logs {
		read, err := witnessLogs.ReadOps(logID)

		if err != nil {
			return "", fmt.Errorf("failed to read log, %v", err)
		}

		chkpt, err := read.GetLatest()

		if err != nil {
			return "", fmt.Errorf("failed to get latest checkpoint, %v", err)
		}

		s += string(chkpt)
	}

	return
}

func witnessCmd(_ *shell.Interface, arg []string) (res string, err error) {
	if witnessLogs != nil {
		return dumpWitnessLogs()
	}

	sec, pub, err := note.GenerateKey(rand.Reader, string(witnessName))

	if err != nil {
		return "", fmt.Errorf("failed to generate derived note key, %v", err)
	}

	signer, err := f_note.NewSignerForCosignatureV1(sec)

	if err != nil {
		return "", fmt.Errorf("failed to create note signer, %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", witnessPort))

	if err != nil {
		return "", fmt.Errorf("failed to listen on port %d, %v", witnessPort, err)
	}

	mf := prometheus.MetricFactory{
		Prefix: "omniwitness_",
	}
	monitoring.SetMetricFactory(mf)

	opConfig := omniwitness.OperatorConfig{
		WitnessKeys:     []note.Signer{signer},
		WitnessVerifier: signer.Verifier(),
		FeedInterval:    30 * time.Second,
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	witnessLogs = NewPersistence()

	log.Printf("starting omniwitness on :%d (%s)", witnessPort, pub)

	go func() {
		if err = omniwitness.Main(context.Background(), opConfig, witnessLogs, listener, client); err != nil {
			log.Printf("omniwitness error, %v", err)
		}
	}()

	return
}
