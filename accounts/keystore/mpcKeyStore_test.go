package keystore

import (
	"testing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
)
var testPath = "/home/cranelv/work/test/"
func TestStoreMpcKey(t *testing.T) {
	testKs := NewMpcKeyStore(testPath)
	key1, _  := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	mpcGroup := []uint64{34,35,36}
	_,err := testKs.StoreMpcKey(&key1.PublicKey,key1.D,32,2,mpcGroup,"111111")
	if err != nil {
		t.Error(err)
	}
}
func TestUnlockMpcKey(t *testing.T) {
	testKs := NewMpcKeyStore(testPath)
	var testAddr common.MpcAddress
	for key,_ := range testKs.keyStoreFiles{
		testAddr = key
	}
	err := testKs.UnlockAddress(testAddr,"111111")
	if err != nil {
		t.Error(err)
	}
	mpcKey := testKs.GetMpcKey(testAddr)
	if mpcKey == nil {
		t.Error("unLock error")
	}
	t.Log(common.ToHex(mpcKey.Address[:]))
	t.Log(common.ToHex(mpcKey.PrivateKey.Bytes()))
	t.Log(mpcKey.MPCSeed)
	t.Log(mpcKey.Quorum)
	t.Log(mpcKey.MPCGroup)
}