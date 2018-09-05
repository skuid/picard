package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

/*
BuildRequestWithBody will encode the requestBody, create the new http request
with the method, url, and encoded request body. It will also inject the proxy
headers, set the authorization header, and form the new context.

After all of that, it will return the http request to the caller.
*/
func BuildRequestWithBody(ctx context.Context, requestBody interface{}, httpMethod string, url string, proxyHeaders ProxyHeaders) (*http.Request, error) {
	bytesBuffer := bytes.NewBuffer(nil)
	if requestBody != nil {
		err := json.NewEncoder(bytesBuffer).Encode(requestBody)
		if err != nil {
			return nil, err
		}
	}

	request, err := http.NewRequest(
		httpMethod,
		url,
		bytesBuffer,
	)
	if err != nil {
		return nil, err
	}
	request.Header = InjectProxyHeaders(request.Header, proxyHeaders)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", proxyHeaders.SessionID))

	if ctx != nil {
		request = request.WithContext(ctx)
	}

	return request, nil
}

/*
MakeRequest makes a request to pliny. It will call BuildRequestWithBody to get
the http request object. It will also set the timeout and do the request.

This also has a side effect, in that it will go ahead and write the response from
the call back out to the caller (writing to response.Body).

The status code and any errors will be returned to the caller.
*/
func MakeRequest(ctx context.Context, proxyHeaders ProxyHeaders, url string, httpMethod string, requestBody interface{}, responseObject interface{}) (int, error) {
	sugar := zap.L().Sugar()
	defer sugar.Sync()

	request, err := BuildRequestWithBody(ctx, requestBody, httpMethod, url, proxyHeaders)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	client := &http.Client{}

	cto := viper.GetDuration("client_timeout")
	if cto >= 0 {
		client.Timeout = cto
	}

	response, err := client.Do(request)
	if err != nil {
		sugar.Errorf("Error making %s request to %s. Reason: %v", httpMethod, url, err)
		return http.StatusInternalServerError, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err == io.EOF || len(body) <= 0 {
		sugar.Warnf("Authenticating against %s returned an empty body. Status: %d", url, response.StatusCode)
		return response.StatusCode, nil
	}

	if err != nil {
		sugar.Errorf("Error reading the response body from %s. Error: %v", url, err)
		return http.StatusInternalServerError, nil
	}

	err = json.Unmarshal(body, &responseObject)

	switch {
	case response.StatusCode < 200 || response.StatusCode >= 300:
		sugar.Warnf("Authenticating against %s returned an unsuccessful status of %d. Reason: %v", url, response.StatusCode, string(body))
		return response.StatusCode, nil
	case err != nil:
		// Some other malformed/non-JSON response body...
		sugar.Warnf("Authenticating against %s returned a malformed body. Status: %d. Body: %v", url, response.StatusCode, string(body))
		return http.StatusInternalServerError, err
	}
	return response.StatusCode, nil
}
