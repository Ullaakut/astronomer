package signature

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
)

func Check(report *SignedReport) error {
	data, err := json.Marshal(report.Report)
	if err != nil {
		return fmt.Errorf("unable to marshal trust report: %v", err)
	}

	hashedReport := sha512.Sum512(data)

	pemData, err := ioutil.ReadFile("key.pem")
	if err != nil {
		return fmt.Errorf("unable to find private key: %v", err)
	}

	keyBlock, _ := pem.Decode(pemData)
	if err != nil {
		return fmt.Errorf("unable to decode private key: %v", err)
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("unable to parse private key: %v", err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, hashedReport[:])
	if err != nil {
		return fmt.Errorf("unable to sign trust report: %v", err)
	}

	if !bytes.Equal(signature, report.Signature) {
		return errors.New("signature doesn't match")
	}

	return nil
}
