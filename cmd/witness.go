// Copyright (c) The TamaGo Authors. All Rights Reserved.
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
	"sync"
	"time"

	f_note "github.com/transparency-dev/formats/note"
	"github.com/transparency-dev/witness/monitoring"
	"github.com/transparency-dev/witness/monitoring/prometheus"
	"github.com/transparency-dev/witness/omniwitness"

	"golang.org/x/mod/sumdb/note"

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
func NewPersistence() *inMemoryPersistence {
	return &inMemoryPersistence{
		checkpoints: make(map[string][]byte),
	}
}

type inMemoryPersistence struct {
	// mu allows checkpoints to be read concurrently, but
	// exclusively locked for writing.
	mu          sync.RWMutex
	checkpoints map[string][]byte
}

func (p *inMemoryPersistence) Init(_ context.Context) error {
	return nil
}

func (p *inMemoryPersistence) Logs() ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res := make([]string, 0, len(p.checkpoints))
	for k := range p.checkpoints {
		res = append(res, k)
	}
	return res, nil
}

func (p *inMemoryPersistence) Latest(_ context.Context, logID string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.checkpoints[logID], nil
}

func (p *inMemoryPersistence) Update(_ context.Context, logID string, f func([]byte) ([]byte, error)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	u, err := f(p.checkpoints[logID])
	if err != nil {
		return err
	}

	p.checkpoints[logID] = u
	return nil
}

var witnessLogs *inMemoryPersistence

func dumpWitnessLogs() (s string, err error) {
	logs, err := witnessLogs.Logs()

	if err != nil {
		return "", fmt.Errorf("failed to get log list, %v", err)
	}

	for _, logID := range logs {
		chkpt, err := witnessLogs.Latest(nil, logID)

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
