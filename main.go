package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
	"tailscale.com/tsnet"

	"github.com/creack/pty"
)

func main() {
	authKey := os.Getenv("TS_AUTHKEY") // create an ephemeral, tagged key in admin console
	if authKey == "" {
		log.Fatal("TS_AUTHKEY env var is required")
	}

	s := &tsnet.Server{
		Hostname: "go-ssh-proxy", // shows up in your Tailnet
		AuthKey:  authKey,
		Ephemeral: true,          // auto-cleanup when offline
		Dir:       "./state",     // or tmp; persistent if you want reuse
	}
	defer s.Close()

	ln, err := s.Listen("tcp", ":2222") // Tailnet-only listener
	if err != nil {
		log.Fatalf("tsnet listen: %v", err)
	}
	log.Printf("Tailnet SSH server listening on %s:2222", s.Hostname)

	// Setup SSH server configuration
	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		log.Fatal(err)
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			authorized := os.Getenv("SSH_AUTHORIZED_KEY")
			if authorized == "" {
				return nil, fmt.Errorf("SSH_AUTHORIZED_KEY env var is required")
			}
			parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorized))
			if err != nil {
				return nil, fmt.Errorf("failed to parse authorized key: %v", err)
			}
			// Replace ssh.KeysEqual with byte comparison
			if string(key.Marshal()) == string(parsedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unauthorized public key for %q", conn.User())
		},
	}
	config.AddHostKey(signer)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go func(nc net.Conn) {
			defer nc.Close()
			sshConn, chans, reqs, err := ssh.NewServerConn(nc, config)
			if err != nil {
				log.Printf("failed handshake: %s", err)
				return
			}
			log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
			go ssh.DiscardRequests(reqs)
			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "only session supported")
					continue
				}
				channel, requests, err := newChannel.Accept()
				if err != nil {
					log.Printf("could not accept channel: %v", err)
					continue
				}
				go func(ch ssh.Channel, reqs <-chan *ssh.Request) {
					var ptyRequested bool
					for req := range reqs {
						switch req.Type {
						case "pty-req":
							ptyRequested = true
							req.Reply(true, nil)
						case "shell":
							if len(req.Payload) > 0 {
								req.Reply(false, nil)
								continue
							}
							req.Reply(true, nil)
							cmd := exec.Command("/bin/sh")
							if ptyRequested {
								ptmx, err := pty.Start(cmd)
								if err != nil {
									log.Printf("could not start pty: %v", err)
									return
								}
								go func() { io.Copy(ptmx, ch) }()
								io.Copy(ch, ptmx)
								ptmx.Close()
							} else {
								cmd.Stdin = ch
								cmd.Stdout = ch
								cmd.Stderr = ch
								if err := cmd.Run(); err != nil {
									log.Printf("shell exited with error: %v", err)
								}
							}
							ch.Close()
							return
						default:
							req.Reply(false, nil)
						}
					}
				}(channel, requests)
			}
		}(conn)
	}
}
