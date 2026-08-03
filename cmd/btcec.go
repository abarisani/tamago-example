// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	testKey     = "22a47fa09a223f2aa079edf85a7c2d4f8720ee63e502ee2869afab7de234b80c"
	testMessage = "TamaGo - bare metal Go"
	testTaproot = "bc1pkgpgdhlt5stnz4gn89d3h2vjm3shu90ysuk7dv8guw0ywrwppm9qa2l4qe"
	testHRP     = "bc"
)

// privateKey returns the hardcoded test key.
func privateKey() (privKey *btcec.PrivateKey, err error) {
	b, err := hex.DecodeString(testKey)

	if err != nil {
		return
	}

	privKey, _ = btcec.PrivKeyFromBytes(b)

	return
}

// ecdsaSignature demonstrates ECDSA signing and verification over the
// secp256k1 curve.
func ecdsaSignature(log *log.Logger) (err error) {
	privKey, err := privateKey()

	if err != nil {
		return
	}

	digest := chainhash.DoubleHashB([]byte(testMessage))
	sig := ecdsa.Sign(privKey, digest)

	if !sig.Verify(digest, privKey.PubKey()) {
		return errors.New("signature verification failed")
	}

	// Flipping a bit of the digest must invalidate the signature.
	digest[0] ^= 0x01

	if sig.Verify(digest, privKey.PubKey()) {
		return errors.New("signature verified against modified message")
	}

	log.Printf("  public key: %x", privKey.PubKey().SerializeCompressed())
	log.Printf("   signature: %.31x...", sig.Serialize())

	return
}

// schnorrSignature demonstrates BIP340 schnorr signing and verification.
func schnorrSignature(log *log.Logger) (err error) {
	privKey, err := privateKey()

	if err != nil {
		return
	}

	digest := chainhash.DoubleHashB([]byte(testMessage))

	sig, err := schnorr.Sign(privKey, digest)

	if err != nil {
		return
	}

	pubKey, err := schnorr.ParsePubKey(schnorr.SerializePubKey(privKey.PubKey()))

	if err != nil {
		return
	}

	if !sig.Verify(digest, pubKey) {
		return errors.New("signature verification failed")
	}

	digest[0] ^= 0x01

	if sig.Verify(digest, pubKey) {
		return errors.New("signature verified against modified message")
	}

	log.Printf("  x-only public key: %.28x...", schnorr.SerializePubKey(privKey.PubKey()))
	log.Printf("          signature: %.28x...", sig.Serialize())

	return
}

// taprootAddress demonstrates BIP341 output key derivation, tweaking an
// internal key with a BIP86 (empty script tree) commitment, and encodes the
// result as a BIP350 bech32m address.
func taprootAddress(log *log.Logger) (err error) {
	privKey, err := privateKey()

	if err != nil {
		return
	}

	internalKey, err := schnorr.ParsePubKey(schnorr.SerializePubKey(privKey.PubKey()))

	if err != nil {
		return
	}

	t := chainhash.TaggedHash(chainhash.TagTapTweak, schnorr.SerializePubKey(internalKey))

	var tweak btcec.ModNScalar

	if tweak.SetBytes((*[32]byte)(t)) != 0 {
		return errors.New("tweak out of range")
	}

	var internalPoint, tweakPoint, outputPoint btcec.JacobianPoint

	internalKey.AsJacobian(&internalPoint)
	btcec.ScalarBaseMultNonConst(&tweak, &tweakPoint)
	btcec.AddNonConst(&internalPoint, &tweakPoint, &outputPoint)
	outputPoint.ToAffine()

	outputKey := btcec.NewPublicKey(&outputPoint.X, &outputPoint.Y)

	program, err := bech32.ConvertBits(schnorr.SerializePubKey(outputKey), 8, 5, true)

	if err != nil {
		return
	}

	addr, err := bech32.EncodeM(testHRP, append([]byte{0x01}, program...))

	if err != nil {
		return
	}

	if addr != testTaproot {
		return errors.New("address mismatch, " + addr)
	}

	log.Printf("  internal key: %x", schnorr.SerializePubKey(internalKey))
	log.Printf("    output key: %x", schnorr.SerializePubKey(outputKey))
	log.Printf("       address: %s", addr)

	return
}

// zkProof is a non-interactive zero-knowledge proof of knowledge of the
// discrete logarithm of a secp256k1 point.
type zkProof struct {
	R *btcec.PublicKey // R is the prover commitment, k*G for a random nonce k.
	S btcec.ModNScalar // S is the prover response, k + e*x.
}

// zkChallenge derives the proof challenge, applying the Fiat-Shamir transform
// to the interactive protocol by binding the commitment, the statement and the
// context into a single hash rather than requiring a verifier round trip.
func zkChallenge(r *btcec.PublicKey, p *btcec.PublicKey, ctx []byte) (e btcec.ModNScalar) {
	h := chainhash.TaggedHash(
		[]byte("TamaGoZKPoK"),
		r.SerializeCompressed(),
		p.SerializeCompressed(),
		ctx,
	)

	e.SetBytes((*[32]byte)(h))

	return
}

// zkProve proves knowledge of the private key for the corresponding public
// key, without revealing anything about the key itself.
func zkProve(privKey *btcec.PrivateKey, ctx []byte) (proof *zkProof, err error) {
	nonce, err := btcec.NewPrivateKey()

	if err != nil {
		return
	}

	proof = &zkProof{
		R: nonce.PubKey(),
	}

	e := zkChallenge(proof.R, privKey.PubKey(), ctx)
	proof.S.Mul2(&e, &privKey.Key).Add(&nonce.Key)

	return
}

// zkVerify verifies a proof of knowledge against a public key, learning
// nothing beyond the fact that the prover knows the corresponding private key.
func zkVerify(pubKey *btcec.PublicKey, ctx []byte, proof *zkProof) bool {
	e := zkChallenge(proof.R, pubKey, ctx)

	var lhs, rhs, statement, commitment btcec.JacobianPoint

	btcec.ScalarBaseMultNonConst(&proof.S, &lhs)

	pubKey.AsJacobian(&statement)
	btcec.ScalarMultNonConst(&e, &statement, &rhs)

	proof.R.AsJacobian(&commitment)
	btcec.AddNonConst(&commitment, &rhs, &rhs)

	lhs.ToAffine()
	rhs.ToAffine()

	return lhs.X.Equals(&rhs.X) && lhs.Y.Equals(&rhs.Y)
}

// zeroKnowledgeProof demonstrates a Schnorr sigma protocol, made
// non-interactive through the Fiat-Shamir transform, proving knowledge of a
// secp256k1 private key without disclosing it.
func zeroKnowledgeProof(log *log.Logger) (err error) {
	privKey, err := privateKey()

	if err != nil {
		return
	}

	ctx := []byte("tamago-example")
	proof, err := zkProve(privKey, ctx)

	if err != nil {
		return
	}

	if !zkVerify(privKey.PubKey(), ctx, proof) {
		return errors.New("proof verification failed")
	}

	if zkVerify(privKey.PubKey(), []byte("other context"), proof) {
		return errors.New("proof verified against modified context")
	}

	forged := &zkProof{R: proof.R}
	forged.S.Set(&proof.S).Add(new(btcec.ModNScalar).SetInt(1))

	if zkVerify(privKey.PubKey(), ctx, forged) {
		return errors.New("forged proof verified")
	}

	other, err := btcec.NewPrivateKey()

	if err != nil {
		return
	}

	if zkVerify(other.PubKey(), ctx, proof) {
		return errors.New("proof verified against unrelated public key")
	}

	s := proof.S.Bytes()

	log.Printf("  commitment: %x", proof.R.SerializeCompressed())
	log.Printf("    response: %x", s[:])
	log.Printf("   statement: knowledge of the key for %.19x...", privKey.PubKey().SerializeCompressed())

	return
}

// randomness exercises the entropy source backing key and nonce generation,
// which on bare metal runs is the SoC TRNG.
func randomness(log *log.Logger) (err error) {
	const samples = 4

	var previous []byte

	for i := 0; i < samples; i++ {
		key, err := btcec.NewPrivateKey()

		if err != nil {
			return err
		}

		current := key.Serialize()

		if bytes.Equal(current, previous) {
			return errors.New("generated identical keys")
		}

		previous = current
	}

	log.Printf("  generated %d unique secp256k1 keys", samples)

	return
}

func btcTest() (tag string, res string) {
	var buf strings.Builder

	tag = "btcec"
	l := log.New(&buf, log.Prefix(), 0)

	tests := []struct {
		name string
		test func(*log.Logger) error
	}{
		{"entropy and key generation", randomness},
		{"ECDSA signature (secp256k1)", ecdsaSignature},
		{"schnorr signature (BIP340)", schnorrSignature},
		{"taproot address derivation (BIP341/BIP350)", taprootAddress},
		{"zero knowledge proof of knowledge (Schnorr/Fiat-Shamir)", zeroKnowledgeProof},
	}

	for _, t := range tests {
		l.Printf("%s:", t.name)

		if err := t.test(l); err != nil {
			l.Printf("  FAILED, %v", err)
			continue
		}

		l.Printf("  OK")
	}

	return tag, buf.String()
}
