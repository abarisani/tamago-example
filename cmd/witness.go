// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"sigsum.org/sigsum-go/pkg/crypto"
	"sigsum.org/sigsum-go/pkg/monitor"
	"sigsum.org/sigsum-go/pkg/policy"
	"sigsum.org/sigsum-go/pkg/types"

	"github.com/usbarmory/tamago-example/shell"
)

const (
	policyName    = "sigsum-test1-2025"
	queryInterval = 60 * time.Second
)

var (
	wlogPath = "/witness.log"
	wlog     *log.Logger
	wlogFile *os.File
)

type callbacks struct{}

func (_ callbacks) NewTreeHead(logKeyHash crypto.Hash, signedTreeHead types.SignedTreeHead) {
	wlog.Printf("new %x tree, size %d", logKeyHash, signedTreeHead.Size)
}

func (_ callbacks) NewLeaves(logKeyHash crypto.Hash, numberOfProcessedLeaves uint64, indices []uint64, leaves []types.Leaf) {
	wlog.Printf("new %x leaves, count %d, total processed %d", logKeyHash, len(leaves), numberOfProcessedLeaves)
}

func (_ callbacks) Alert(logKeyHash crypto.Hash, e error) {
	wlog.Printf("alert log %x, %v", logKeyHash, e)
}

func init() {
	shell.Add(shell.Cmd{
		Name: "witness",
		Help: "start/inspect sigsum monitor",
		Fn:   witnessCmd,
	})
}

func witnessCmd(_ *shell.Interface, arg []string) (res string, err error) {
	if wlog != nil {
		return catCmd(nil, []string{wlogPath})
	}

	if wlogFile, err = os.OpenFile(wlogPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return
	}
	wlog = log.New(wlogFile, "", 0)

	pub, _, err := crypto.NewKeyPair()

	if err != nil {
		return "", fmt.Errorf("failed to generate key pair, %v", err)
	}

	config := monitor.Config{
		QueryInterval: queryInterval,
		Callbacks:     callbacks{},
	}

	policy, err := policy.ByName(policyName)

	if err != nil {
		return "", fmt.Errorf("failed to load policy %s, %v", policyName, err)
	}

	log.Printf("starting sigsum monitor (%x)", pub)
	monitor.StartMonitoring(context.TODO(), policy, &config, nil)

	return
}
