package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	httpRequestTimeout    = 5 * time.Second
	serverShutdownTimeout = 100 * time.Millisecond
)

var (
	gateway             string
	listenAddress       string
	shouldInitiateLogin bool

	//go:embed login.html
	loginPageHTML string
)

func init() {
	gatewayFlag := flag.String(
		"gateway",
		"",
		"<host>[:<port>] of the VPN gateway. Required.",
	)
	initiateLoginFlag := flag.Bool(
		"initiate-login",
		false,
		"Try to initiate the login process automatically.",
	)
	listenPortFlag := flag.Int(
		"listen-port",
		8020,
		"Where to receive the local SAML request.",
	)

	flag.Parse()

	if *gatewayFlag == "" {
		flag.Usage()
		os.Exit(2)
	}
	gateway = *gatewayFlag

	listenAddress = net.JoinHostPort("127.0.0.1", strconv.Itoa(*listenPortFlag))

	shouldInitiateLogin = *initiateLoginFlag
}

func run() error {
	idChannel := make(chan string)
	errorChannel := make(chan error)

	http.HandleFunc("GET /{$}", func(response http.ResponseWriter, request *http.Request) {
		idChannel <- request.URL.Query().Get("id")
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(response, loginPageHTML)
	})

	server := &http.Server{Addr: listenAddress}
	{
		listener, err := net.Listen("tcp4", listenAddress)
		if err != nil {
			return fmt.Errorf("net.Listen error: %v", err)
		}
		go func() {
			if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
				errorChannel <- fmt.Errorf("HTTP server error: %v", err)
			}
		}()
	}

	{
		url := fmt.Sprintf("https://%s/remote/saml/start?redirect=1", gateway)
		fmt.Fprintln(os.Stderr, "To login, please open", url)
		if shouldInitiateLogin {
			tryInitiateLogin(url)
		}
	}

	var id string
	select {
	case id = <-idChannel:
		break
	case err := <-errorChannel:
		return err
	}

	{
		timeoutContext, cancelTimeout := context.WithTimeout(
			context.Background(),
			serverShutdownTimeout,
		)
		defer cancelTimeout()

		if err := server.Shutdown(timeoutContext); err != nil {
			fmt.Fprintln(os.Stderr, "HTTP server shutdown error:", err)
		}
	}

	if id == "" {
		return errors.New("missing id parameter")
	}
	cookie, err := getCookie(id)
	if err != nil {
		return err
	}
	fmt.Println(cookie.Value)

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
