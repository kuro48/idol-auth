package http

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type cfAccessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type cfJWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type cfJWKS struct {
	Keys []cfJWK `json:"keys"`
}

type cfAccessVerifier struct {
	certsURL   string
	audience   string
	mu         sync.RWMutex
	keyCache   map[string]*rsa.PublicKey
	cachedAt   time.Time
	cacheTTL   time.Duration
	httpClient *http.Client
}

func newCFAccessVerifier(teamDomain, audience string) *cfAccessVerifier {
	return &cfAccessVerifier{
		certsURL:   "https://" + teamDomain + "/cdn-cgi/access/certs",
		audience:   audience,
		keyCache:   make(map[string]*rsa.PublicKey),
		cacheTTL:   time.Hour,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (v *cfAccessVerifier) Verify(rawToken string) (string, error) {
	claims := &cfAccessClaims{}
	_, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, _ := token.Header["kid"].(string)
		key, err := v.getKey(kid)
		if err != nil {
			// Refresh and retry once when key is not found or stale.
			if refreshErr := v.fetchKeys(); refreshErr != nil {
				return nil, fmt.Errorf("refresh cf certs: %w", refreshErr)
			}
			key, err = v.getKey(kid)
			if err != nil {
				return nil, err
			}
		}
		return key, nil
	}, jwt.WithAudience(v.audience))

	if err != nil {
		return "", fmt.Errorf("cf access jwt invalid: %w", err)
	}

	if claims.Email == "" {
		return "", fmt.Errorf("cf access jwt missing email claim")
	}

	return claims.Email, nil
}

func (v *cfAccessVerifier) getKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keyCache[kid]
	stale := time.Since(v.cachedAt) > v.cacheTTL
	v.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}

	if err := v.fetchKeys(); err != nil {
		if ok {
			// Return cached key on transient network failure rather than blocking all admin access.
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	key, ok = v.keyCache[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cf certs: key not found for kid %q", kid)
	}
	return key, nil
}

func (v *cfAccessVerifier) fetchKeys() error {
	resp, err := v.httpClient.Get(v.certsURL)
	if err != nil {
		return fmt.Errorf("fetch cf certs: %w", err)
	}
	defer resp.Body.Close()

	var jwks cfJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode cf certs: %w", err)
	}

	cache := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.N == "" || k.E == "" {
			continue
		}
		key, err := rsaKeyFromJWK(k)
		if err != nil {
			continue
		}
		cache[k.Kid] = key
	}

	v.mu.Lock()
	v.keyCache = cache
	v.cachedAt = time.Now()
	v.mu.Unlock()

	return nil
}

func rsaKeyFromJWK(k cfJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}
