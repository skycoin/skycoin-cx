//go:build !nocipher

package cipher

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cxhash"
	"github.com/skycoin/skycoin/src/cipher"
)

// Signatures made by cipher.SignHash of Skycoin verify, and a changed hash,
// signature or key, or one of the wrong length, does not.
func TestCipherSignatureVerifySkycoinSignatures(t *testing.T) {
	for i := range 8 {
		pubKey, secKey := cipher.GenerateKeyPair()
		otherPubKey, _ := cipher.GenerateKeyPair()
		digest := cipher.SHA256(cxhash.CxHashOfBytes([]byte{byte(i), 'c', 'x'}))
		signature := cipher.MustSignHash(digest, secKey)

		if !CipherSignatureVerify(digest[:], signature[:], pubKey[:]) {
			t.Fatalf("pair %d: a Skycoin signature does not verify", i)
		}
		if CipherSignatureVerify(digest[:], signature[:], otherPubKey[:]) {
			t.Errorf("pair %d: verifies with another key", i)
		}
		changedDigest := digest
		changedDigest[0] ^= 1
		if CipherSignatureVerify(changedDigest[:], signature[:], pubKey[:]) {
			t.Errorf("pair %d: verifies another hash", i)
		}
		changedSignature := signature
		changedSignature[5] ^= 1
		if CipherSignatureVerify(digest[:], changedSignature[:], pubKey[:]) {
			t.Errorf("pair %d: a changed signature verifies", i)
		}
		if CipherSignatureVerify(digest[:31], signature[:], pubKey[:]) || CipherSignatureVerify(digest[:], signature[:64], pubKey[:]) || CipherSignatureVerify(digest[:], signature[:], pubKey[:32]) {
			t.Errorf("pair %d: an input of the wrong length verifies", i)
		}
		if CipherSignatureVerify(nil, nil, nil) {
			t.Errorf("empty inputs verify")
		}
	}
}

// CipherSignatureVerify allocates only what the Skycoin call allocates.
func TestCipherSignatureVerifyAllocs(t *testing.T) {
	pubKey, secKey := cipher.GenerateKeyPair()
	digest := cipher.SHA256(cxhash.CxHashOfBytes([]byte("abc")))
	signature := cipher.MustSignHash(digest, secKey)
	skycoin := testing.AllocsPerRun(10, func() { cipher.VerifyPubKeySignedHash(pubKey, signature, digest) })
	views := testing.AllocsPerRun(10, func() { CipherSignatureVerify(digest[:], signature[:], pubKey[:]) })
	if views != skycoin {
		t.Errorf("%.0f allocations per verify, the Skycoin call makes %.0f", views, skycoin)
	}
}

func BenchmarkCipherSignatureVerify(b *testing.B) {
	pubKey, secKey := cipher.GenerateKeyPair()
	digest := cipher.SHA256(cxhash.CxHashOfBytes([]byte("abc")))
	signature := cipher.MustSignHash(digest, secKey)
	b.ReportAllocs()
	for b.Loop() {
		CipherSignatureVerify(digest[:], signature[:], pubKey[:])
	}
}
