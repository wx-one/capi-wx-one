package controller

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type WxOneClients struct {
	httpClient    *resty.Client
	graphqlClient graphql.Client
}

func setCookiesMiddleware(next http.RoundTripper, cookie string) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Add the Cookie header to the request
		req.Header.Add("Cookie", cookie)
		return next.RoundTrip(req)
	})
}

// Session return the DO session.
func NewWXOneClients(log logr.Logger, ctx context.Context) (WxOneClients, error) {
	host := os.Getenv("WX_ONE_HOST")
	username := os.Getenv("WX_ONE_USERNAME")
	password := os.Getenv("WX_ONE_PASSWORD")

	log.Info("Creating WX-ONE clients")

	restClient := resty.New()
	restClient.SetCookieJar(nil)
	restClient.SetTimeout(10 * time.Second)

	challengeResponse, err := restClient.R().SetBody(map[string]string{"username": username}).
		Post(fmt.Sprintf("%s/challenge", host))
	if err != nil {
		log.Error(
			err,
			"An unexpected error occurred when creating the WX-ONE API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WX-ONE Client Error: "+err.Error(),
		)
		return WxOneClients{}, err
	}

	var challenge map[string]interface{}
	err = json.Unmarshal(challengeResponse.Body(), &challenge)
	if err != nil {
		log.Error(
			err,
			"An unexpected error occurred when creating the WX-ONE API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WX-ONE Client Error: "+err.Error(),
		)
		return WxOneClients{}, err
	}

	hash := sha512.Sum512([]byte(strings.ToUpper(password) + challenge["salt"].(string)))
	hashValue := challenge["challenge"].(string) + challenge["date"].(string) + "wizardtales.com" + hex.EncodeToString(hash[:])
	hash = sha512.Sum512([]byte(hashValue))

	// Define the number of rounds
	rounds := int(challenge["rounds"].(float64))

	// Perform the hashing multiple rounds
	for i := 1; i < rounds; i++ {
		// Concatenate your strings
		hashValue := challenge["challenge"].(string) + challenge["date"].(string) + "wizardtales.com" + hex.EncodeToString(hash[:])
		hash = sha512.Sum512([]byte(hashValue))
	}

	// Convert the final hash to a hex string
	hashedPassword := hex.EncodeToString(hash[:])

	loginResponse, err := restClient.R().SetBody(map[string]string{"username": username, "password": hashedPassword}).
		Post(fmt.Sprintf("%s/login", host))
	if err != nil {
		log.Error(
			err,
			"An unexpected error occurred when creating the WX-ONE API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WX-ONE Client Error: "+err.Error(),
		)
		return WxOneClients{}, err
	}

	var login map[string]interface{}
	err = json.Unmarshal(loginResponse.Body(), &login)
	if err != nil {
		log.Error(
			err,
			"An unexpected error occurred when creating the WX-ONE API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WX-ONE Client Error: "+err.Error(),
		)
		return WxOneClients{}, err
	}

	// Extract cookies from the response
	cookies := loginResponse.Header().Get("Set-Cookie")
	cookieParts := strings.Split(cookies, ";")
	cookie := cookieParts[0]

	httpClient := &http.Client{
		Transport: setCookiesMiddleware(http.DefaultTransport, cookie),
	}

	grqphqlClient := graphql.NewClient(host+"/graphql", httpClient)
	_, meErr := me(ctx, grqphqlClient)

	if meErr != nil {
		log.Error(
			err,
			"An unexpected error occurred when creating the WX-ONE API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WX-ONE Client Error: "+meErr.Error(),
		)
		return WxOneClients{}, err
	}

	wxOneClients := WxOneClients{
		httpClient:    restClient,
		graphqlClient: grqphqlClient,
	}

	return wxOneClients, err
}
