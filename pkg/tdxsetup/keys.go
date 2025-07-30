package tdxsetup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

type KeyInitializerer interface {
	WaitForKey(ctx context.Context) (string, error)
}

type WebServerKeyInitializer struct {
	URL string
}

func (w *WebServerKeyInitializer) WaitForKey(ctx context.Context) (string, error) {
	keyReceivedChan := make(chan string)
	serverErrChan := make(chan error)

	server := &http.Server{
		Addr: w.URL,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprint(w, "Only POST method is allowed")
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Error reading request: %v", err)
				return
			}

			key := string(body)
			matched, _ := regexp.MatchString(`^[A-Za-z0-9+/]{68}$`, key)
			if !matched {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "Invalid key format, expected base64-encoded OpenSSH ed25519 public key")
				return
			}

			keyReceivedChan <- key
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Key received and stored successfully")
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		if err := server.Shutdown(context.Background()); err != nil {
			return "", fmt.Errorf("server shutdown error: %w", err)
		}
		return "", ctx.Err()
	case err := <-serverErrChan:
		return "", fmt.Errorf("server error: %w", err)
	case key := <-keyReceivedChan:
		if err := server.Shutdown(context.Background()); err != nil {
			return "", fmt.Errorf("server shutdown error: %w", err)
		}
		return key, nil
	}
}
