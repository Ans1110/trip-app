package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
)

const jwkKeyID = "tripapp-key-1"

func buildJWKS(pub *rsa.PublicKey) *JWKResponse {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	return &JWKResponse{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: jwkKeyID,
				Alg: "RS256",
				N:   n,
				E:   e,
			},
		},
	}
}
