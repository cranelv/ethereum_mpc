package step

import (
	"crypto/ecdsa"
	"math/big"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
	manCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type TxSign_CalSignStep struct {
	TXSign_Lagrange_Step
}

func CreateTxSign_CalSignStep(result protocol.MpcResultInterface,nodeinfo protocol.MpcNodeInterface, resultKey string) *TxSign_CalSignStep {
	log.Info("CreateTxSign_CalSignStep begin")

	signSeedKeys := []string{protocol.MpcTxSignSeed}
	resultKeys := []string{resultKey}
	mpc := &TxSign_CalSignStep{*CreateTXSign_Lagrange_Step(result,nodeinfo, signSeedKeys, resultKeys)}
	return mpc
}

func (txStep *TxSign_CalSignStep) InitStep() error {
	log.Info("TxSign_CalSignStep.InitStep begin")

	privateKey, err := txStep.mpcResult.GetValue(protocol.MpcPrivateShare)
	if err != nil {
		return err
	}
	ar, err := txStep.mpcResult.GetValue(protocol.MpcSignARResult)
	if err != nil {
		return err
	}
//	log.Error("ARVALUE2","ar",ar.(*big.Int))
	aPoint, err := txStep.mpcResult.GetValue(protocol.MpcSignAPoint)
	if err != nil {
		return err
	}

	r, err := txStep.mpcResult.GetValue(protocol.MpcSignR)
	if err != nil {
		return err
	}

	c, err := txStep.mpcResult.GetValue(protocol.MpcSignC)
	if err != nil {
		return err
	}

	txHash, err := txStep.mpcResult.GetValue(protocol.MpcTxHash)
	if err != nil {
		return err
	}

	arInv := new(big.Int).Set(ar.(*big.Int))
	arInv.ModInverse(arInv, crypto.Secp256k1N)
	invRPoint := new(ecdsa.PublicKey)
	invRPoint.Curve = secp256k1.S256()
	pointA := aPoint.([2]*big.Int)
	invRPoint.X, invRPoint.Y = secp256k1.S256().ScalarMult(pointA[0], pointA[1], arInv.Bytes())
	if invRPoint.X == nil || invRPoint.Y == nil {
		log.Error("TxSign_CalSignStep.InitStep, invalid r point")
		return protocol.ErrPointZero
	}

//	log.Error("TxSign_CalSignStep.InitStep","InvR-X",invRPoint.X,"InvR-Y", invRPoint.Y)
	SignSeed := new(big.Int).Set(invRPoint.X)
	SignSeed.Mod(SignSeed, crypto.Secp256k1N)
	var v uint64
	if invRPoint.X.Cmp(SignSeed) == 0 {
		v = 0
	} else {
		v = 2
	}

	invRPoint.Y.Mod(invRPoint.Y, big.NewInt(2))
	if invRPoint.Y.Cmp(big.NewInt(0)) != 0 {
		v |= 1
	}

	txStep.mpcResult.SetValue(protocol.MpcTxSignResultR, new(big.Int).Set(SignSeed))
//	log.Error("TxSign_CalSignStep.InitStep","ResultR", SignSeed)
	txStep.mpcResult.SetValue(protocol.MpcTxSignResultV,v)
	priKey := privateKey.(*big.Int)
	SignSeed.Mul(SignSeed, priKey)
	SignSeed.Mod(SignSeed, crypto.Secp256k1N)
	hash := new(big.Int).SetBytes(txHash.([]byte))
	SignSeed.Add(SignSeed, hash)
	SignSeed.Mod(SignSeed, crypto.Secp256k1N)
	SignSeed.Mul(SignSeed, r.(*big.Int))
	SignSeed.Mod(SignSeed, crypto.Secp256k1N)
	SignSeed.Add(SignSeed, c.(*big.Int))
	SignSeed.Mod(SignSeed, crypto.Secp256k1N)

	txStep.mpcResult.SetValue(protocol.MpcTxSignSeed ,SignSeed )

	err = txStep.TXSign_Lagrange_Step.InitStep()
	if err != nil {
		log.Info("TxSign_CalSignStep.InitStep, initStep fail, err:%s", err.Error())
		return err
	} else {
		log.Info("TxSign_CalSignStep.InitStep succeed")
		return nil
	}
}

func (txStep *TxSign_CalSignStep) FinishStep( mpc protocol.MpcManager) error {
	log.Info("TxSign_CalSignStep.FinishStep begin")

	err := txStep.TXSign_Lagrange_Step.FinishStep(mpc)
	if err != nil {
		return err
	}

	R, err := txStep.mpcResult.GetValue(protocol.MpcTxSignResultR)
	if err != nil {
		log.Error("MpcDistributor.SignTransaction, GetValue fail.","key", protocol.MpcTxSignResultR)
		return err
	}
	V, err := txStep.mpcResult.GetValue(protocol.MpcTxSignResultV)
	if err != nil {
		log.Error("MpcDistributor.SignTransaction, GetValue fail.","key", protocol.MpcTxSignResultV)
		return err
	}

	S, err := txStep.mpcResult.GetValue(protocol.MpcTxSignResult)
	if err != nil {
		log.Error("MpcDistributor.SignTransaction, GetValue fail.","key", protocol.MpcTxSignResult)
		return err
	}
	txHash, err := txStep.mpcResult.GetValue(protocol.MpcTxHash)
	if err != nil {
		return err
	}
	from, err := txStep.mpcResult.GetValue(protocol.MpcAddress)
	if err != nil {
		return err
	}

	err = mpc.SignTransaction(txStep.mpcResult)
	if err != nil {
		return err
	}
	sign,err := crypto.TransSignature(R.(*big.Int),S.(*big.Int),V.(uint64))
	if err != nil {
		return err
	}
//	log.Error("TxSign_CalSignStep","calSign",common.ToHex(sign))
	hash := common.BytesToHash(txHash.([]byte))
	pub, err := manCrypto.Ecrecover(hash[:], sign)
//	log.Error("TxSign_CalSignStep","calSign",common.ToHex(sign),"hash",hash.String())
	if err != nil {
		return err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return errors.New("invalid public key")
	}
	pubKey := manCrypto.ToECDSAPub(pub)
	addr := keystore.PubkeyToMpcAddress(*pubKey)
	fromAddr := common.BytesToMpcAddress(from.([]byte))
	if fromAddr != addr{
		log.Error("TxSign_CalSignStep Error","hash",hash,"from",fromAddr,"Sign",addr)
	}
	txStep.mpcResult.SetValue(protocol.MpcContextResult,sign)
	log.Info("TxSign_CalSignStep.FinishStep succeed")
	return nil
}
