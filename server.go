package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/armon/go-socks5"
	"github.com/caarlos0/env/v6"
)

type params struct {
	User              string        `env:"PROXY_USER" envDefault:""`
	Password          string        `env:"PROXY_PASSWORD" envDefault:""`
	Port              string        `env:"PROXY_PORT" envDefault:"1080"`
	AllowedDestFqdn   string        `env:"ALLOWED_DEST_FQDN" envDefault:""`
	ConnectionTimeout time.Duration `env:"CONNECTION_TIMEOUT"`
}

// ListenerWithTimeout wraps a net.Listener and sets a deadline for each connection.
type ListenerWithTimeout struct {
	net.Listener
	timeout time.Duration
}

// Accept waits for and returns the next connection to the listener.
func (l *ListenerWithTimeout) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}
	// Set the deadline for the connection
	err = conn.SetDeadline(time.Now().Add(l.timeout))
	if err != nil {
		return conn, err
	}
	return conn, nil
}

// Close closes the listener.
func (l *ListenerWithTimeout) Close() error {
	return l.Listener.Close()
}

// Addr returns the listener's network address.
func (l *ListenerWithTimeout) Addr() net.Addr {
	return l.Listener.Addr()
}

func main() {
	// Working with app params
	cfg := params{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Printf("%+v\n", err)
	}

	// Initialize socks5 config
	socks5conf := &socks5.Config{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	if cfg.User+cfg.Password != "" {
		creds := socks5.StaticCredentials{
			os.Getenv("PROXY_USER"): os.Getenv("PROXY_PASSWORD"),
		}
		cator := socks5.UserPassAuthenticator{Credentials: creds}
		socks5conf.AuthMethods = []socks5.Authenticator{cator}
	}

	if cfg.AllowedDestFqdn != "" {
		socks5conf.Rules = PermitDestAddrPattern(cfg.AllowedDestFqdn)
	}

	server, err := socks5.New(socks5conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Start listening proxy service on port %s\n", cfg.Port)
	l, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.ConnectionTimeout > 0 {
		l = &ListenerWithTimeout{
			Listener: l,
			timeout:  cfg.ConnectionTimeout,
		}
	}

	if err := server.Serve(l); err != nil {
		log.Fatal(err)
	}
}
