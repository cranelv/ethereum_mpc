package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"crypto/elliptic"
	"io"
)
var (
	Secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	Secp256k1halfN = new(big.Int).Div(Secp256k1N, big.NewInt(2))
	one = new(big.Int).SetInt64(1)
)
func UintRand(MaxValue uint64) (uint64, error) {
	num, err := rand.Int(rand.Reader, new(big.Int).SetUint64(MaxValue))
	if err != nil {
		return 0, err
	}

	return num.Uint64(), nil
}
// randFieldElement returns a random element of the field underlying the given
// curve using the procedure given in [NSA] A.2.1.
func RandFieldElement(c elliptic.Curve,) (k *big.Int, err error) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

func GetRandCoefficients(num int) ([]*big.Int, error) {
	randCoefficient := make([]*big.Int, num)
	for i := 0; i < num; i++ {
		key, err := RandFieldElement(crypto.S256())
		if err != nil {
			return nil, err
		}
		randCoefficient[i] = key
	}

	return randCoefficient, nil
}

func EvaluatePoly(coefficient []*big.Int, x *big.Int) *big.Int {
	degree := len(coefficient) - 1
	sumx := make([]big.Int, degree+1)
	for i := 0; i <= degree; i++ {
		sumx[i].Set(coefficient[i])
	}

	for i := 1; i <= degree; i++ {
		for j := i; j <= degree; j++ {
			sumx[j].Mul(&sumx[j], x)
			sumx[j].Mod(&sumx[j], Secp256k1N)
		}
	}

	sum := big.NewInt(0)
	for i := 0; i < len(sumx); i++ {
		sum.Add(sum, &sumx[i])
		sum.Mod(sum, Secp256k1N)
	}

	return sum
}

func evaluateB(x []*big.Int) []*big.Int {
	k := len(x)
	b := make([]*big.Int, k)
	for i := 0; i < k; i++ {
		b[i] = evaluateb(x, i)
	}

	return b
}

func evaluateb(x []*big.Int, i int) *big.Int {
	k := len(x)
	sum := big.NewInt(1)
	temp1 := big.NewInt(1)
	temp2 := big.NewInt(1)
	for j := 0; j < k; j++ {
		if j != i {
			temp1.Sub(x[j], x[i])
			temp1.ModInverse(temp1, Secp256k1N)
			temp2.Mul(x[j], temp1)
			sum.Mul(sum, temp2)
			sum.Mod(sum, Secp256k1N)
		} else {
			continue
		}
	}

	return sum
}

// Lagrange's polynomial interpolation algorithm
func Lagrange(f []*big.Int, x []*big.Int) *big.Int {
	degree := len(x) - 1
	b := evaluateB(x)
	s := big.NewInt(0)
	temp1 := big.NewInt(1)

	for i := 0; i < degree+1; i++ {
		temp1.Mul(f[i], b[i])
		s.Add(s, temp1)
		s.Mod(s, Secp256k1N)
	}

	return s
}

func TransSignature(R *big.Int, S *big.Int, V uint64) ([]byte, error) {
	if S.Cmp(new(big.Int).Div(Secp256k1N, big.NewInt(2))) > 0 {
		V ^= 1
		S.Sub(Secp256k1N,S)
	}

	sig := make([]byte, 65)
	copy(sig[:], math.PaddedBigBytes(R, 32))
	copy(sig[32:], math.PaddedBigBytes(S, 32))
	sig[64] = byte(V)
	return sig, nil
}

func SenderEcrecover(sighash, sig []byte) (common.Address, error) {
	log.Info("SenderEcrecover, sigHash:%s, sig:%s", common.ToHex(sighash), common.ToHex(sig))

	pub, err := crypto.Ecrecover(sighash, sig)
	if err != nil {
		log.Error("SenderEcrecover, crypto Ecrecover fail, err:%s", err.Error())
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		log.Error("SenderEcrecover, pub's value isn't zero in first byte")
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

func ValidatePrivateKey(k *big.Int) bool {
	if k == nil || k.Sign() == 0 {
		return false
	}

	return true
}
