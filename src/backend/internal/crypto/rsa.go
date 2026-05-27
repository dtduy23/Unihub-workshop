package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

type RSAProvider struct {
	privateKey *rsa.PrivateKey
}

func NewRSAProvider(privateKeyPEM string) (*RSAProvider, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Thử parse PKCS8 nếu PKCS1 lỗi
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
	}

	return &RSAProvider{privateKey: priv}, nil
}

// TicketPayload là cấu trúc dữ liệu bên trong mã QR (Khớp 100% với Mobile)
type TicketPayload struct {
	StudentID  string `json:"sid"`
	UserID     string `json:"uid"`
	WorkshopID string `json:"wid"`
	Signature  string `json:"sig"`
}

// SignTicket tạo chuỗi JSON đã ký RSA cho vé (4 trường)
func (p *RSAProvider) SignTicket(studentID, userID, workshopID string) (string, error) {
	// Chuỗi dữ liệu thô để ký: sid|uid|wid
	rawData := fmt.Sprintf("%s|%s|%s", studentID, userID, workshopID)
	hashed := sha256.Sum256([]byte(rawData))

	signature, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign ticket: %v", err)
	}

	sigBase64 := base64.StdEncoding.EncodeToString(signature)

	// Tạo JSON hoàn chỉnh cho QR
	payload := TicketPayload{
		StudentID:  studentID,
		UserID:     userID,
		WorkshopID: workshopID,
		Signature:  sigBase64,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// GetPublicKeyPEM trả về Public Key định dạng PEM để gửi cho Client/Mobile verify
func (p *RSAProvider) GetPublicKeyPEM() (string, error) {
	pubKey := &p.privateKey.PublicKey
	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	return string(pem.EncodeToMemory(block)), nil
}
