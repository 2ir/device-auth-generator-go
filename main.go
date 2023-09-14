package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
)



func getPublicAccessToken() (string, error) {
    url := "https://account-public-service-prod.ol.epicgames.com/account/api/oauth/token"
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": "basic OThmN2U0MmMyZTNhNGY4NmE3NGViNDNmYmI0MWVkMzk6MGEyNDQ5YTItMDAxYS00NTFlLWFmZWMtM2U4MTI5MDFjNGQ3",
	}
	data := "grant_type=client_credentials"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return "", err
	}
	accessToken := jsonResponse["access_token"].(string)
	return accessToken, nil
}


func getDeviceCode(accessToken string) (string, string, error) {
    url := "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/deviceAuthorization"
    headers := map[string]string{
        "Authorization": fmt.Sprintf("bearer %s", accessToken),
        "Content-Type": "application/x-www-form-urlencoded",
    }
    req, err := http.NewRequest("POST", url, nil)
    if err != nil {
        return "", "", err
    }
    for key, value := range headers {
        req.Header.Set(key, value)
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", "", err
    }
    var jsonResponse map[string]interface{}
    err = json.Unmarshal(body, &jsonResponse)
    if err != nil {
        return "", "", err
    }
    deviceCode := jsonResponse["device_code"].(string)
    verificationUriComplete := jsonResponse["verification_uri_complete"].(string)
    return deviceCode, verificationUriComplete, nil
}


func getAccessToken(deviceCode string) (string, string, error) {
    url_ := "https://account-public-service-prod.ol.epicgames.com/account/api/oauth/token"
    headers := map[string]string{
        "Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": "basic OThmN2U0MmMyZTNhNGY4NmE3NGViNDNmYmI0MWVkMzk6MGEyNDQ5YTItMDAxYS00NTFlLWFmZWMtM2U4MTI5MDFjNGQ3",
    }
    data := url.Values{}
    data.Set("grant_type", "device_code")
    data.Set("device_code", deviceCode)
    req, err := http.NewRequest("POST", url_, bytes.NewBufferString(data.Encode()))
    if err != nil {
        return "", "", err
    }
    for key, value := range headers {
        req.Header.Set(key, value)
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil  {
        return "", "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
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
        return "", "", fmt.Errorf("Missing access token and account id in response")
    }
    return accessToken, accountId, nil
}



func getDeviceAuth(accountId, accessToken string) (string, string, string, error) {
    url := fmt.Sprintf("https://account-public-service-prod.ol.epicgames.com/account/api/public/account/%s/deviceAuth", accountId)
    headers := map[string]string{
        "Authorization": fmt.Sprintf("Bearer %s", accessToken),
    }
    req, err := http.NewRequest("POST", url, nil)
    if err != nil {
        return "", "", "", err
    }
    for key, value := range headers {
        req.Header.Set(key, value)
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", "", "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
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
        return "", "", "", fmt.Errorf("Missing account id, device id and secret values in the json response")
    }
    return accountIdResp, deviceIdResp, secret, nil
    
}



func main() {
    var accessToken, accountId string
	accessToken, err := getPublicAccessToken()
	if err != nil {
		log.Fatalf("Error getting token: %v", err)
	}
    deviceCode, verificationUriComplete, err := getDeviceCode(accessToken)
    fmt.Println("[<] " + verificationUriComplete)
    for {
        accessToken, accountId, err = getAccessToken(deviceCode)
        if err == nil && accessToken != "" && accountId != "" {
            break
        }
    }
    accountId, deviceId, secret, err := getDeviceAuth(accountId, accessToken)
    fmt.Println("[<] account id: " + accountId)
    fmt.Println("[<] device id: " + deviceId)
    fmt.Println("[<] secret: ", secret)
}
