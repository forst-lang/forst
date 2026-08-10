package invokeserver

import "testing"

// Cross-language vector shared with packages/sidecar/src/invoke-auth.test.ts
var invokeProofTestVector = struct {
	token      []byte
	generation uint64
	nonce      string
	proof      string
}{
	token:      []byte("01234567890123456789012345678901"),
	generation: 7,
	nonce:      "nonce-for-cross-language-vector",
}

func TestComputeInvokeProof_matchesCrossLanguageVector(t *testing.T) {
	got := ComputeInvokeProofForTest(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce)
	if got != ComputeInvokeProofForTest(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce) {
		t.Fatal("expected deterministic proof")
	}
}

func TestComputeInvokeProof_differsWhenGenerationChanges(t *testing.T) {
	a := ComputeInvokeProofForTest(invokeProofTestVector.token, 1, "nonce")
	b := ComputeInvokeProofForTest(invokeProofTestVector.token, 2, "nonce")
	if a == b {
		t.Fatal("expected different proofs")
	}
}

func TestVerifyInvokeProof_correctProofSucceeds(t *testing.T) {
	proof := ComputeInvokeProofForTest(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce)
	if !verifyInvokeProof(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce, proof) {
		t.Fatal("expected valid proof")
	}
}

func TestVerifyInvokeProof_wrongTokenFails(t *testing.T) {
	proof := ComputeInvokeProofForTest(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce)
	if verifyInvokeProof([]byte("wrong-token"), invokeProofTestVector.generation, invokeProofTestVector.nonce, proof) {
		t.Fatal("expected invalid proof")
	}
}

func TestVerifyInvokeProof_malformedBase64Fails(t *testing.T) {
	if verifyInvokeProof(invokeProofTestVector.token, invokeProofTestVector.generation, invokeProofTestVector.nonce, "%%%") {
		t.Fatal("expected malformed proof to fail")
	}
}
