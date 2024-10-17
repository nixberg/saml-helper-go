package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

func getCookie(id string) (*http.Cookie, error) {
	timeoutContext, cancelTimeout := context.WithTimeout(
		context.Background(),
		httpRequestTimeout,
	)
	defer cancelTimeout()

	request, err := http.NewRequestWithContext(
		timeoutContext,
		"GET",
		fmt.Sprintf("https://%s/remote/saml/auth_id?id=%s", gateway, id),
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := new(http.Client).Do(request)
	if err != nil {
		return nil, err
	}
	response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`got response status "%s"`, response.Status)
	}

	cookies := response.Cookies()

	index := slices.IndexFunc(cookies, func(cookie *http.Cookie) bool {
		return strings.HasPrefix(cookie.Name, "SVPNCOOKIE")
	})
	if index == -1 {
		return nil, errors.New("SVPNCOOKIE missing from response")
	}

	return cookies[index], nil
}
