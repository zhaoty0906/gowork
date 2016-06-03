package libkademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
	mathrand "math/rand"
	"sss"
	"time"
)

type VanashingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
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

func CalculateSharedKeyLocations1(accessKey int64, count int64) (ids []ID) {

	currentTime := time.Now()
	h := currentTime.Hour()
	m := currentTime.Minute()

	y := currentTime.Year()
	mt := currentTime.Month()
	d := currentTime.Day()
	l := currentTime.Location()

	randSource := time.Date(y, mt, d, h, m, 0, 0, l)
	rand := mathrand.New(mathrand.NewSource(randSource.UnixNano()))
	newKey := rand.Int63()
	accessKey = newKey * accessKey
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

const (
	waitTime = 1
	unit     = 60
)

func (k *Kademlia) VanishData(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	ekey := GenerateRandomCryptoKey()
	cdata := encrypt(ekey, data)

	//break keys into pieces and stored in a map brokenkeys
	brokenkeys, err := sss.Split(numberKeys, threshold, ekey)
	if err != nil {
		log.Print("Error when break encrypt key")
	}

	accessKey := GenerateRandomAccessKey()
	keysloc := CalculateSharedKeyLocations(accessKey, int64(numberKeys))

	//store keys in the map
	for i := 0; i < len(keysloc); i++ {

		ke := byte(i + 1)
		v := brokenkeys[ke]
		all := append([]byte{ke}, v...)
		k.DoIterativeStore(keysloc[i], all)

	}

	vdo.AccessKey = accessKey
	vdo.Ciphertext = cdata
	vdo.NumberKeys = numberKeys
	vdo.Threshold = threshold

	go func() {
		for {
			if timeoutSeconds > 0 {
				time.Sleep(waitTime * unit * time.Second)
				timeoutSeconds -= waitTime
				id := CalculateSharedKeyLocations1(accessKey, int64(numberKeys))
				for i := 0; i < len(id); i++ {

					ke := byte(i + 1)
					v := brokenkeys[ke]
					all := append([]byte{ke}, v...)
					k.DoIterativeStore(id[i], all)

				}
			} else {
				break
			}
		}
	}()

	return
}

func (k *Kademlia) UnvanishData(vdo VanashingDataObject) (data []byte) {
	// get the location of N sharedKeys
	ids := CalculateSharedKeyLocations(vdo.AccessKey, int64(vdo.NumberKeys))
	// get the sharedKeys
	var sharedKeys map[byte][]byte
	for _, id := range ids {
		var foundKeys []byte
		foundKeys, err := k.DoIterativeFindValue(id)
		if err != nil {
			log.Fatal("UnvanishData IterativeFindValue Error")
		}
		sharedKeys[foundKeys[0]] = foundKeys[1:]
	}
	// get the key K
	var mainKey []byte
	mainKey = sss.Combine(sharedKeys)
	// get the original text
	text := decrypt(mainKey, vdo.Ciphertext)

	return text
}
