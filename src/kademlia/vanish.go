package kademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	mathrand "math/rand"
	"sss"
	"time"
)

type VanashingDataObject struct {
	// comments by haomin
	AccessKey  int64  // L in project description
	Ciphertext []byte // C
	NumberKeys byte   // N
	Threshold  byte   // T

	// a local copy of N pieces for testing purpose, by haomin
	LocalCopy [][]byte // 1~N as key, pieces of byte arrays as value
}

func GenerateRandomCryptoKey() (ret []byte) { // return K
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) { // return L
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func VanishData(kadem Kademlia, data []byte, numberKeys byte,
	threshold byte) (vdo VanashingDataObject) {
	K := GenerateRandomCryptoKey()
	C := encrypt(K, data)
	N := numberKeys
	T := threshold
	shares, err := sss.Split(N, T, K)
	if err != nil {
		fmt.Printf("err!!!!")
		panic(err)
	}
	L := GenerateRandomAccessKey()
	lc := make([][]byte, int(N))
	vdo = VanashingDataObject{L, C, N, T, lc}
	//indices := CalculateSharedKeyLocations(L, int64(N)) // where the key pieces to be stored
	for k := 0; k < int(N); k++ {
		tmp_k := []byte{byte(k + 1)}
		tmp_v := shares[byte(k+1)]

		all := append(tmp_k, tmp_v...)
		lc[k] = all
		//fmt.Printf("debugging")
		//(&kadem).DoIterativeStore(indices[k], all)
	}
	return
}

func UnvanishData(kadem Kademlia, vdo VanashingDataObject) (data []byte) {
	//L := vdo.AccessKey
	C := vdo.Ciphertext
	N := vdo.NumberKeys
	T := vdo.Threshold
	//indices := CalculateSharedKeyLocations(L, int64(N)) // where the key pieces are stored

	count := 0
	shares := make(map[byte][]byte, T) // the pieces we need to re-construct our key

	for k := 0; k < int(N); k++ {
		//Bytes, Contacts := (&kadem).DoIterativeFindValue_Internal(indices[k])
		//if len(Contacts) == 0 { // nothing found :-(
		//	continue
		//}
		fmt.Printf("%v\n", k)
		Bytes := vdo.LocalCopy[k]

		all := Bytes
		tmp_k := all[0]
		tmp_v := all[1:]
		shares[tmp_k] = tmp_v
		count++
		//if count >= int(T) {
		//	break
		//}
	}
	if count >= int(T) { // enough!
		K := sss.Combine(shares)
		return decrypt(K, C)
	} else { // failed to collect enough pieces
		return []byte("not enough!!!")
	}
}
