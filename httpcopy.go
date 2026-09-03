package apitest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var divider = strings.Repeat("-", 10)
var requestDebugPrefix = fmt.Sprintf("%s>", divider)
var responseDebugPrefix = fmt.Sprintf("<%s", divider)

func debugLog(prefix, header, msg string) {
	fmt.Printf("\n%s %s\n%s\n", prefix, header, msg)
}

func copyHttpResponse(response *http.Response) *http.Response {
	if response == nil {
		return nil
	}

	var resBodyBytes []byte
	if response.Body != nil {
		resBodyBytes, _ = io.ReadAll(response.Body)
		response.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
	}

	resCopy := &http.Response{
		Header:        map[string][]string{},
		StatusCode:    response.StatusCode,
		Status:        response.Status,
		Body:          io.NopCloser(bytes.NewBuffer(resBodyBytes)),
		Proto:         response.Proto,
		ProtoMinor:    response.ProtoMinor,
		ProtoMajor:    response.ProtoMajor,
		ContentLength: response.ContentLength,
	}

	for name, values := range response.Header {
		resCopy.Header[name] = append([]string(nil), values...)
	}

	return resCopy
}

func copyHttpRequest(request *http.Request) *http.Request {
	resCopy := &http.Request{
		Method:        request.Method,
		Host:          request.Host,
		Proto:         request.Proto,
		ProtoMinor:    request.ProtoMinor,
		ProtoMajor:    request.ProtoMajor,
		ContentLength: request.ContentLength,
		RemoteAddr:    request.RemoteAddr,
	}
	resCopy = resCopy.WithContext(request.Context())

	if request.Body != nil {
		bodyBytes, _ := io.ReadAll(request.Body)
		resCopy.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if request.URL != nil {
		r2URL := new(url.URL)
		*r2URL = *request.URL
		resCopy.URL = r2URL
	}

	headers := make(http.Header)
	for k, values := range request.Header {
		for _, hValue := range values {
			headers.Add(k, hValue)
		}
	}
	resCopy.Header = headers

	return resCopy
}
