package server

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ullaakut/astronomer/pkg/trust"
	"github.com/ullaakut/disgo"
)

type SignedReport struct {
	*trust.Report

	Signature []byte
}

func SendReport(report *trust.Report) error {
	data, err := json.Marshal(report)
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

	return sendReport(SignedReport{
		Report:    report,
		Signature: signature,
	})
}

func sendReport(report SignedReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("unable to marshal signed report: %v", err)
	}

	response, err := http.Post("http://localhost", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("unable to send signed report to astronomer server: %v", err)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("astronomer server did not trust this report: %v", response.Status)
	}

	disgo.Debugln("Signed report successfully sent to astronomer server, thanks for your contribution!")

	return nil
}
