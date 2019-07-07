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
)

// Check verifies whether a report has been signed
// with a legitimate version of Astronomer.
func Check(report *SignedReport) error {
	data, err := json.Marshal(report.Report)
	if err != nil {
		return fmt.Errorf("unable to marshal trust report: %v", err)
	}

	hashedReport := sha512.Sum512(data)

	keyBlock, _ := pem.Decode([]byte(pemData))
	if keyBlock == nil {
		return fmt.Errorf("unable to decode private key")
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
