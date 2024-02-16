package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	tokenURL                 = "https://account-public-service-prod.ol.epicgames.com/account/api/oauth/token"
	deviceAuthorizationURL   = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/deviceAuthorization"
	contentType              = "application/x-www-form-urlencoded"
	authorizationHeaderValue = "Basic OThmN2U0MmMyZTNhNGY4NmE3NGViNDNmYmI0MWVkMzk6MGEyNDQ5YTItMDAxYS00NTFlLWFmZWMtM2U4MTI5MDFjNGQ3"
	httpClientTimeout        = 10 * time.Second
)

var httpClient = &http.Client{
	Timeout: httpClientTimeout,
}

func getPublicAccessToken(ctx context.Context) (string, error) {
	data := "grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return "", err
	}

	accessToken, ok := jsonResponse["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access_token not found in response")
	}

	return accessToken, nil
}

func getDeviceCode(ctx context.Context, accessToken string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", deviceAuthorizationURL, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", contentType)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return "", "", err
	}

	deviceCode, ok1 := jsonResponse["device_code"].(string)
	verificationUriComplete, ok2 := jsonResponse["verification_uri_complete"].(string)
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("required fields not found in the response")
	}

	return deviceCode, verificationUriComplete, nil
}

func getAccessToken(ctx context.Context, deviceCode string) (string, string, error) {
	data := url.Values{}
	data.Set("grant_type", "device_code")
	data.Set("device_code", deviceCode)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return "", "", err
	}

	accessToken, ok1 := jsonResponse["access_token"].(string)
	accountId, ok2 := jsonResponse["account_id"].(string)
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("required fields not found in the response")
	}

	return accessToken, accountId, nil
}

func getDeviceAuth(ctx context.Context, accountId, accessToken string) (string, string, string, error) {
	url := fmt.Sprintf("https://account-public-service-prod.ol.epicgames.com/account/api/public/account/%s/deviceAuth", accountId)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return "", "", "", err
	}

	accountIdResp, ok1 := jsonResponse["accountId"].(string)
	deviceIdResp, ok2 := jsonResponse["deviceId"].(string)
	secret, ok3 := jsonResponse["secret"].(string)
	if !ok1 || !ok2 || !ok3 {
		return "", "", "", fmt.Errorf("required fields not found in the response")
	}

	return accountIdResp, deviceIdResp, secret, nil
}

func main() {
	ctx := context.Background()

	accessToken, err := getPublicAccessToken(ctx)
	if err != nil {
		log.Fatalf("Error getting public access token: %v", err)
	}

	deviceCode, verificationUriComplete, err := getDeviceCode(ctx, accessToken)
	if err != nil {
		log.Fatalf("Error getting device code: %v", err)
	}
	fmt.Printf("Please authorize your device here: %s\n", verificationUriComplete)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var accessTokenFinal, accountIdFinal string
	for {
		select {
		case <-ctx.Done():
			log.Fatal("Authorization timed out. Please try again.")
		case <-ticker.C:
			accessTokenFinal, accountIdFinal, err = getAccessToken(ctx, deviceCode)
			if err == nil && accessTokenFinal != "" && accountIdFinal != "" {
				fmt.Println("Authorization successful")
				goto Authorized
			}
		}
	}

Authorized:
	accountId, deviceId, secret, err := getDeviceAuth(ctx, accountIdFinal, accessTokenFinal)
	if err != nil {
		log.Fatalf("Error registering device: %v", err)
	}
	fmt.Printf("Device registered successfully:\nAccount ID: %s\nDevice ID: %s\nSecret: %s\n", accountId, deviceId, secret)
}
