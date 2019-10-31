package keystore

import (
	"math/big"
	"github.com/pborman/uuid"
	"github.com/ethereum/go-ethereum/common"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
	"golang.org/x/crypto/scrypt"
	"github.com/ethereum/go-ethereum/common/math"
	"errors"
	"sync"
	"path/filepath"
	"io/ioutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)
var MpcKeyStoreScheme = "mpcKeystore"
type MpcKey struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.MpcAddress
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *big.Int
	//MPC
	MPCSeed uint64
	Quorum  uint64
	MPCGroup []uint64

}
type encryptedMpcKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	MPCSeed uint64	   `json:"mpcseed"`
	Quorum  uint64	   `json:"quorum"`
	MPCGroup []uint64	   `json:"mpcGroup"`
	Version int        `json:"version"`
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func (key* MpcKey)EncryptKey(auth string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(auth)
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(key.PrivateKey, 32)

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	encryptedKeyJSONV3 := encryptedMpcKeyJSONV3{
		hex.EncodeToString(key.Address[:]),
		cryptoStruct,
		key.Id.String(),
		key.MPCSeed,
		key.Quorum,
		key.MPCGroup,
		version,
	}
	return json.Marshal(encryptedKeyJSONV3)
}
func PubkeyToMpcAddress(p ecdsa.PublicKey) common.MpcAddress{
	var addr common.MpcAddress
	copy(addr[:],secp256k1.CompressPubkey(p.X,p.Y))
	return addr
}
func newMpcKey(pKey *ecdsa.PublicKey, pShare *big.Int, seed uint64,quorum uint64,MPCGroup[]uint64) (*MpcKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = crypto.S256()
	priv.D = pShare
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(pShare.Bytes())
	id := uuid.NewRandom()

	key := &MpcKey{
		Id:         id,
//		Address:    common.BytesToHash()secp256k1.CompressPubkey(pKey.X,pKey.Y),
		PrivateKey: pShare,
		MPCSeed:seed,
		Quorum:quorum,
		MPCGroup:MPCGroup,
	}
	copy(key.Address[:],secp256k1.CompressPubkey(pKey.X,pKey.Y))
	//3 bytes for every seed
	return key , nil
}
// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptMpcKey(keyjson []byte, auth string) (*MpcKey, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyId []byte
		err             error
	)
	if version, ok := m["version"].(string); ok && version != "3" {
		return nil,errors.New("version must equal 3")
	}
	k := new(encryptedMpcKeyJSONV3)
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, err
	}
	k1 := encryptedKeyJSONV3{
		k.Address,
		k.Crypto,
		k.Id,
		k.Version,
	}
	keyBytes, keyId, err = decryptKeyV3(&k1, auth)
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}
	key := crypto.ToECDSAUnsafe(keyBytes)

	mpcKey := MpcKey{
		Id:         uuid.UUID(keyId),
//		Address:    common.HexToAddress(k.Address),
		PrivateKey: key.D,
		MPCSeed : k.MPCSeed,
		Quorum: k.Quorum,
		MPCGroup:k.MPCGroup,
	}
	copy(mpcKey.Address[:],common.FromHex(k.Address))
	return &mpcKey,nil
}
//cranelv add storemanKey
func storeMpcKey(ks keyStore, pKey *ecdsa.PublicKey, pShare *big.Int, seed uint64,quorum uint64,MPCGroup []uint64, passphrase string)(*MpcKey,accounts.MpcAccount,  error){
	key, err := newMpcKey(pKey, pShare, seed,quorum,MPCGroup)
	if err!= nil {
		return nil,accounts.MpcAccount{}, err
	}

	a := accounts.MpcAccount{key.Address,accounts.URL{Scheme: MpcKeyStoreScheme, Path: ks.JoinPath(mpcKeyFileName(key.Address))}}
	if err := ks.StoreMpcKey(a.URL.Path, key, passphrase); err != nil {
		key.PrivateKey = big.NewInt(0)
		return nil,a, err
	}

	return key,a, err
}
type MpcKeyStore struct {
	mu sync.RWMutex
	keyDir string
	storage  keyStore                     // Storage backend, might be cleartext or encrypted
	keyStoreFiles map[common.MpcAddress] string
	unlocked map[common.MpcAddress]*MpcKey // Currently unlocked account (decrypted private keys)
}
func (mks* MpcKeyStore) UnlockAddress(address common.MpcAddress,password string)error{
	mks.mu.Lock()
	defer mks.mu.Unlock()
	if _,exist := mks.unlocked[address];exist{
		return nil
	}
	if file,exist := mks.keyStoreFiles[address];!exist{
		return ErrNoMatch
	}else{
		key,err := mks.storage.GetMpcKey(address,file,password)
		if err != nil{
			return err
		}
		mks.unlocked[address] = key
		return nil
	}
}
func (mks* MpcKeyStore) init(keydir string){
	mks.keyDir = keydir
	err := mks.scan()
	if err != nil{
		log.Error("MpcKeyStore init Error","error",err)
	}
}

// scan performs a new scan on the given directory, compares against the already
// cached filenames, and returns file sets: creates, deletes, updates.
func (mks* MpcKeyStore) scan() error {

	// List all the failes from the keystore folder
	files, err := ioutil.ReadDir(mks.keyDir)
	if err != nil {
		return err
	}

	for _, fi := range files {
		// Skip any non-key files from the folder
		path := filepath.Join(mks.keyDir, fi.Name())
		if skipKeyFile(fi) {
			log.Trace("Ignoring file on account scan", "path", path)
			continue
		}
		keyjson, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		k := new(encryptedMpcKeyJSONV3)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return err
		}
		mks.keyStoreFiles[common.HexToMpcAddress(k.Address)] = path
	}
	return nil
}

func (mks* MpcKeyStore) GetMpcKey(address common.MpcAddress)*MpcKey{
	mks.mu.RLock()
	defer mks.mu.RUnlock()
	return mks.unlocked[address]
}
func (mks* MpcKeyStore)StoreMpcKey(pKey *ecdsa.PublicKey, pShare *big.Int, seed uint64,quorum uint64,mpcGroup []uint64, passphrase string)(common.MpcAddress,error){
	_,acc,err := storeMpcKey(mks.storage,pKey,pShare,seed,quorum,mpcGroup,passphrase)
	if err != nil{
		return common.MpcAddress{},err
	}
	mks.mu.Lock()
	defer mks.mu.Unlock()
	mks.keyStoreFiles[acc.Address] = acc.URL.Path
	return acc.Address,nil
}
func NewMpcKeyStore(keydir string) *MpcKeyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &MpcKeyStore{storage: &keyStorePassphrase{keydir, LightScryptN, LightScryptP},
	keyStoreFiles:make(map[common.MpcAddress] string),
	unlocked:make(map[common.MpcAddress]*MpcKey)}
	ks.init(keydir)
	return ks
}