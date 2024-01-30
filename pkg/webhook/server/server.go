package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
)

type WebhookServer struct {
	Settings  *Settings
	Namespace string
	Router    chi.Router
}

// Start runs https server to listen for AdmissionReview requests from the API-server.
func (s *WebhookServer) Start() error {
	// Load server certificate.
	certWatcher, err := certwatcher.New(
		s.Settings.ServerCertPath,
		s.Settings.ServerKeyPath,
	)
	if err != nil {
		return fmt.Errorf("load TLS certs: %v", err)
	}

	go func() {
		if err := certWatcher.Start(context.TODO()); err != nil {
			log.Errorf("Unable to watch cert: %v", err)
			// Stop process if server can't start.
			os.Exit(1)
		}
	}()

	// Construct a hostname for certificate.
	host := fmt.Sprintf("%s.%s",
		s.Settings.ServiceName,
		s.Namespace,
	)

	tlsConf := &tls.Config{
		GetCertificate: certWatcher.GetCertificate,
		ServerName:     host,
	}

	// Load client CA if defined
	if len(s.Settings.ClientCAPaths) > 0 {
		roots := x509.NewCertPool()

		for _, caPath := range s.Settings.ClientCAPaths {
			caBytes, err := os.ReadFile(caPath)
			if err != nil {
				return fmt.Errorf("load client CA '%s': %v", caPath, err)
			}

			ok := roots.AppendCertsFromPEM(caBytes)
			if !ok {
				return fmt.Errorf("parse client CA '%s': %v", caPath, err)
			}
		}

		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConf.ClientCAs = roots
	}

	listenAddr := net.JoinHostPort(s.Settings.ListenAddr, s.Settings.ListenPort)
	// Check if port is available
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("try listen on '%s': %v", listenAddr, err)
	}

	timeout := time.Duration(10) * time.Second

	srv := &http.Server{
		Handler:           s.Router,
		TLSConfig:         tlsConf,
		Addr:              listenAddr,
		IdleTimeout:       timeout,
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	go func() {
		log.Infof("Webhook server listens on %s", listenAddr)
		err := srv.ServeTLS(listener, "", "")
		if err != nil {
			log.Errorf("Error starting Webhook https server: %v", err)
			// Stop process if server can't start.
			os.Exit(1)
		}
	}()

	return nil
}
