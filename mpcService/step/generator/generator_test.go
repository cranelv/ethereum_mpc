package generator

import (
	"math/big"
	"testing"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func makeZero(degree, peerNum int, peerInfo []protocol.PeerInfo) ([]big.Int, *big.Int) {
	jrss := make([]RandomPolynomialValue, peerNum)
	for i := 0; i < peerNum; i++ {
		jrss[i] = *CreateJRSSValue(degree, peerNum,"text")
		jrss[i].Initialize(peerInfo,nil)
	}

	lagResult := big.NewInt(0)
	for i := 0; i < peerNum; i++ {
		for j := 0; j < len(jrss[i].PolyValue); j++ {
			jrss[i].PolyValue[j] = big.NewInt(0)
		}
		lagResult.Add(lagResult, jrss[i].RandCoefficient[0])
	}

	lagResult.Mod(lagResult, crypto.Secp256k1N)
	jrssResult := make([]big.Int, peerNum)
	for i := 0; i < peerNum; i++ {
		for j := 0; j < peerNum; j++ {
			jrssResult[i].Add(&jrssResult[i], jrss[j].PolyValue[i])
		}
		jrssResult[i].Mod(&jrssResult[i], crypto.Secp256k1N)
	}

	return jrssResult, lagResult
}

func makeJRSS(degree, peerNum int, peerInfo []protocol.PeerInfo) ([]big.Int, *big.Int) {
	jrss := make([]RandomPolynomialValue, peerNum)
	for i := 0; i < peerNum; i++ {
		jrss[i] = *CreateJRSSValue(degree, peerNum,"test")
	}

	lagResult := big.NewInt(0)
	for i := 0; i < peerNum; i++ {
		jrss[i].Initialize(peerInfo, nil)
		lagResult.Add(lagResult, jrss[i].RandCoefficient[0])
	}

	lagResult.Mod(lagResult, crypto.Secp256k1N)
	jrssResult := make([]big.Int, peerNum)
	for i := 0; i < peerNum; i++ {
		for j := 0; j < peerNum; j++ {
			jrssResult[i].Add(&jrssResult[i], jrss[j].PolyValue[i])
		}
		jrssResult[i].Mod(&jrssResult[i], crypto.Secp256k1N)
	}

	return jrssResult, lagResult
}

func makeJZSS(degree, peerNum int, peerInfo []protocol.PeerInfo) ([]big.Int, *big.Int) {
	jrss := make([]RandomPolynomialValue, peerNum)
	for i := 0; i < peerNum; i++ {
		jrss[i] = *CreateJZSSValue(degree, peerNum,"test")
	}

	lagResult := big.NewInt(0)
	for i := 0; i < peerNum; i++ {
		jrss[i].Initialize(peerInfo, nil)
		lagResult.Add(lagResult, jrss[i].RandCoefficient[0])
	}

	lagResult.Mod(lagResult, crypto.Secp256k1N)
	jrssResult := make([]big.Int, peerNum)
	for i := 0; i < peerNum; i++ {
		for j := 0; j < peerNum; j++ {
			jrssResult[i].Add(&jrssResult[i], jrss[j].PolyValue[i])
		}
		jrssResult[i].Mod(&jrssResult[i], crypto.Secp256k1N)
	}

	return jrssResult, lagResult
}

func makePeerInfo(peerNum int) []protocol.PeerInfo {
	peerInfo := make([]protocol.PeerInfo, peerNum)
	for i := 0; i < peerNum; i++ {
		peerInfo[i] = protocol.PeerInfo{PeerID:new(discover.NodeID), Seed:uint64(i + 2)}
	}

	return peerInfo
}

func TestJRSSZeroMult(t *testing.T) {
	degree := 8
	peerNum := 21
	peerInfo := makePeerInfo(peerNum)
	jrssResult, lagResult := makeZero(degree, peerNum, peerInfo)
	t.Logf("lagResult is: %v", lagResult)
	jrssResult1, lagResult1 := makeZero(degree, peerNum, peerInfo)
	t.Logf("lagResult1 is: %v", lagResult1)
	lagResult.Mul(lagResult, lagResult1)
	lagResult.Mod(lagResult, crypto.Secp256k1N)
	t.Logf("Mult lagResult is: %v", lagResult)
	jzssResult, zResult := makeZero(degree*2, peerNum, peerInfo)
	t.Logf("zero Result is: %v", zResult)
	jrssResultMul := make([]big.Int, peerNum)
	for i := 0; i < peerNum; i++ {
		jrssResultMul[i].Mul(&jrssResult[i], &jrssResult1[i])
		jrssResultMul[i].Add(&jrssResultMul[i], &jzssResult[i])
		jrssResultMul[i].Mod(&jrssResultMul[i], crypto.Secp256k1N)
	}

	for i := 0; i < peerNum-degree*2-1; i++ {
		seedLen := degree*2 + 1
		seed := make([]int, seedLen)
		for j := 0; j < seedLen; j++ {
			seed[j] = i + j
		}

		lag1 := testLagrange(seed, peerInfo, jrssResultMul)
		t.Logf("lagResult 1 is: %v", lag1)
		if lagResult.Cmp(lag1) != 0 {
			t.Error("Lagrange is Error")
		}
	}
}

func TestJRSSGeneratorMult(t *testing.T) {
	degree := 8
	peerNum := 21
	peerInfo := makePeerInfo(peerNum)
	jrssResult, lagResult := makeJRSS(degree, peerNum, peerInfo)
	t.Logf("lagResult is: %v", lagResult)
	jrssResult1, lagResult1 := makeJRSS(degree, peerNum, peerInfo)
	t.Logf("lagResult1 is: %v", lagResult1)
	lagResult.Mul(lagResult, lagResult1)
	lagResult.Mod(lagResult, crypto.Secp256k1N)
	t.Logf("Mult lagResult is: %v", lagResult)
	jzssResult, zResult := makeJZSS(degree*2, peerNum, peerInfo)
	t.Logf("zero Result is: %v", zResult)
	jrssResultMul := make([]big.Int, peerNum)
	for i := 0; i < peerNum; i++ {
		jrssResultMul[i].Mul(&jrssResult[i], &jrssResult1[i])
		jrssResultMul[i].Add(&jrssResultMul[i], &jzssResult[i])
		jrssResultMul[i].Mod(&jrssResultMul[i], crypto.Secp256k1N)
	}

	for i := 0; i < peerNum-degree*2-1; i++ {
		seedLen := degree*2 + 1
		seed := make([]int, seedLen)
		for j := 0; j < seedLen; j++ {
			seed[j] = i + j
		}

		lag1 := testLagrange(seed, peerInfo, jrssResultMul)
		t.Logf("lagResult 1 is: %v", lag1)
		if lagResult.Cmp(lag1) != 0 {
			t.Error("Lagrange is Error")
		}
	}

	for i := 0; i < peerNum-degree*2-1; i++ {
		seedLen := degree * 2
		seed := make([]int, seedLen)
		for j := 0; j < seedLen; j++ {
			seed[j] = i + j
		}

		lag1 := testLagrange(seed, peerInfo, jrssResultMul)
		t.Logf("lagResult 2 is: %v", lag1)
		if lagResult.Cmp(lag1) == 0 {
			t.Error("Lagrange is Error")
		}
	}
}

func TestJRSSGenerator21(t *testing.T) {
	degree := 8
	peerNum := 21
	peerInfo := makePeerInfo(peerNum)
	jrssResult, lagResult := makeJRSS(degree, peerNum, peerInfo)

	t.Logf("lagResult is: %v", lagResult)
	for i := 0; i < peerNum-degree-1; i++ {
		seed := make([]int, degree+1)
		for j := 0; j < degree+1; j++ {
			seed[j] = i + j
		}

		lag1 := testLagrange(seed, peerInfo, jrssResult)
		t.Logf("lagResult 1 is: %v", lag1)
		if lagResult.Cmp(lag1) != 0 {
			t.Error("Lagrange is Error")
		}
	}

}

func TestJRSSGenerator(t *testing.T) {
	jrss1 := CreateJRSSValue(1, 3,"test")
	jrss2 := CreateJRSSValue(1, 3,"test")
	jrss3 := CreateJRSSValue(1, 3,"test")

	peerInfo := []protocol.PeerInfo{
		protocol.PeerInfo{PeerID:new(discover.NodeID), Seed:1},
		protocol.PeerInfo{PeerID:new(discover.NodeID), Seed:2},
		protocol.PeerInfo{PeerID:new(discover.NodeID), Seed:3}}
	jrss1.Initialize(peerInfo, nil)
	jrss2.Initialize(peerInfo, nil)
	jrss3.Initialize(peerInfo, nil)

	jrssResult := make([]big.Int, 3)
	for i := 0; i < 3; i++ {
		jrssResult[i].Add(&jrssResult[i], jrss1.PolyValue[i])
		jrssResult[i].Add(&jrssResult[i], jrss2.PolyValue[i])
		jrssResult[i].Add(&jrssResult[i], jrss3.PolyValue[i])
		jrssResult[i].Mod(&jrssResult[i], crypto.Secp256k1N)
	}

	lagResult := jrss1.RandCoefficient[0]
	lagResult.Add(lagResult, jrss2.RandCoefficient[0])
	lagResult.Add(lagResult, jrss3.RandCoefficient[0])
	lagResult.Mod(lagResult, crypto.Secp256k1N)
	t.Logf("lagResult is: %v", lagResult)
	lag1 := testLagrange([]int{0, 1}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}

	lag1 = testLagrange([]int{0, 2}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}

	lag1 = testLagrange([]int{1, 2}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}

	lag1 = testLagrange([]int{2, 0}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}

	lag1 = testLagrange([]int{2, 1}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}

	lag1 = testLagrange([]int{1, 0}, peerInfo, jrssResult)
	t.Logf("lagResult 1 is: %v", lag1)
	if lagResult.Cmp(lag1) != 0 {
		t.Error("Lagrange is Error")
	}
}

func testLagrange(seed []int, peerInfo []protocol.PeerInfo, jrssResult []big.Int) *big.Int {
	fx := make([]*big.Int, len(seed))
	x := make([]*big.Int, len(seed))
	for i := 0; i < len(seed); i++ {
		x[i] = new(big.Int).SetUint64(peerInfo[seed[i]].Seed)
		fx[i] = &jrssResult[seed[i]]
	}

	return crypto.Lagrange(fx, x)
}
