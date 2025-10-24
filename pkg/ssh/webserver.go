package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
)

type WebServerProvider struct {
	ServerURL string
}

func NewWebServerProvider(serverURL string) *WebServerProvider {
	return &WebServerProvider{
		ServerURL: serverURL,
	}
}

func (w *WebServerProvider) WaitForKey(ctx context.Context) (string, error) {
	keyReceivedChan := make(chan string)
	serverErrChan := make(chan error)

	server := &http.Server{
		Addr: w.ServerURL,
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
			fmt.Fprint(w, "SSH key received and stored successfully")
		}),
	}

	log.Printf("Starting web server on %s to receive SSH key", w.ServerURL)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		server.Shutdown(context.Background())
		return "", ctx.Err()
	case err := <-serverErrChan:
		return "", fmt.Errorf("server error: %w", err)
	case key := <-keyReceivedChan:
		server.Shutdown(context.Background())
		return key, nil
	}
}