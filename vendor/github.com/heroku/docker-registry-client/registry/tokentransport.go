package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type TokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
	Token     string
}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	keepBody := false
	if req.Body != nil {
		//keepBody = !strings.Contains(req.URL.Path, "/blobs/uploads/")
		//	b := req.Body
		//	req.Body = nil
		keepBody = false
		if keepBody {
			bodyBytes, _ = ioutil.ReadAll(req.Body)
		}
	}
	if keepBody {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	var resp *http.Response
	var err error
	if req.Method == "PUT" {
		req2, _ := http.NewRequest("PUT", req.URL.String(), nil)
		req2.Header = req.Header
		resp, err = t.Transport.RoundTrip(req2)
		//if resp == nil && resp.StatusCode != 401 {
		if resp != nil && resp.StatusCode != 401 {
			resp, err = t.Transport.RoundTrip(req)
		}
		if resp == nil {
			resp, err = t.Transport.RoundTrip(req)
		}
		//if strings.Contains(req.URL.Path, "/blobs/uploads") {
		//	resp, err = t.Transport.RoundTrip(req)
		//}

	} else {
		resp, err = t.Transport.RoundTrip(req)
	}

	if keepBody {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if err != nil {
		return resp, err
	}
	if authService := isTokenDemand(resp); authService != nil {
		resp, err = t.authAndRetry(authService, req)
	}
	return resp, err
}

type authToken struct {
	Token string `json:"token"`
}

func (t *TokenTransport) authAndRetry(authService *authService, req *http.Request) (*http.Response, error) {
	token, authResp, err := t.auth(authService)
	if err != nil {
		return authResp, err
	}

	retryResp, err := t.retry(req, token)
	return retryResp, err
}

func (t *TokenTransport) auth(authService *authService) (string, *http.Response, error) {
	authReq, err := authService.Request(t.Username, t.Password)
	if err != nil {
		return "", nil, err
	}

	client := http.Client{
		Transport: t.Transport,
	}

	response, err := client.Do(authReq)
	if err != nil {
		return "", nil, err
	}

	if response.StatusCode != http.StatusOK {
		return "", response, err
	}
	defer response.Body.Close()

	var authToken authToken
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&authToken)
	if err != nil {
		return "", nil, err
	}

	return authToken.Token, nil, nil
}

func (t *TokenTransport) retry(req *http.Request, token string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := t.Transport.RoundTrip(req)
	return resp, err
}

type authService struct {
	Realm   string
	Service string
	Scope   string
}

func (authService *authService) Request(username, password string) (*http.Request, error) {
	url, err := url.Parse(authService.Realm)
	if err != nil {
		return nil, err
	}

	q := url.Query()
	q.Set("service", authService.Service)
	if authService.Scope != "" {
		q.Set("scope", authService.Scope)
	}
	url.RawQuery = q.Encode()

	request, err := http.NewRequest("GET", url.String(), nil)

	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}

	return request, err
}

func isTokenDemand(resp *http.Response) *authService {
	if resp == nil {
		return nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return nil
	}
	return parseOauthHeader(resp)
}

func parseOauthHeader(resp *http.Response) *authService {
	challenges := parseAuthHeader(resp.Header)
	for _, challenge := range challenges {
		if challenge.Scheme == "bearer" {
			return &authService{
				Realm:   challenge.Parameters["realm"],
				Service: challenge.Parameters["service"],
				Scope:   challenge.Parameters["scope"],
			}
		}
	}
	return nil
}
