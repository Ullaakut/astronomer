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
	"fmt"
	"net/http"

	"github.com/ullaakut/astronomer/pkg/context"
	"github.com/ullaakut/astronomer/pkg/trust"
	"github.com/ullaakut/disgo"
)

type SignedReport struct {
	*trust.Report

	RepositoryOwner string
	RepositoryName  string

	Signature []byte
}

func SendReport(ctx *context.Context, report *trust.Report) error {
	signature, err := signReport(report)
	if err != nil {
		return err
	}

	return sendReport(SignedReport{
		Report:          report,
		RepositoryOwner: ctx.RepoOwner,
		RepositoryName:  ctx.RepoName,
		Signature:       signature,
	})
}

func signReport(report *trust.Report) ([]byte, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal trust report: %v", err)
	}

	hashedReport := sha512.Sum512(data)

	keyBlock, rest := pem.Decode([]byte(pemData))
	if len(rest) != 0 {
		return nil, fmt.Errorf("unable to decode private key: %s", pemData)
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, hashedReport[:])
	if err != nil {
		return nil, fmt.Errorf("unable to sign trust report: %v", err)
	}

	return signature, nil
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

var pemData = `👀`
