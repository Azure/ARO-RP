package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	sshTimeout = time.Hour
)

func (s *ssh) Run() error {
	go func() {
		defer recover.Panic(s.log)

		for {
			c, err := s.l.Accept()
			if err != nil {
				return
			}

			go func() {
				defer recover.Panic(s.log)

				err := s.newConn(context.Background(), c)
				if err != nil {
					s.log.Warn(err)
				}
			}()
		}
	}()

	return nil
}

func (s *ssh) newConn(ctx context.Context, c1 net.Conn) error {
	defer c1.Close()

	config := &cryptossh.ServerConfig{}
	*config = *s.baseServerConfig

	var portalDoc *api.PortalDocument
	var connmetadata cryptossh.ConnMetadata

	config.PasswordCallback = func(_connmetadata cryptossh.ConnMetadata, pw []byte) (*cryptossh.Permissions, error) {
		connmetadata = _connmetadata

		password, err := uuid.FromString(string(pw))
		if err != nil {
			return nil, fmt.Errorf("invalid username") // don't echo password attempt to logs
		}

		portalDoc, err = s.dbPortal.Get(ctx, password.String())
		if err != nil {
			return nil, fmt.Errorf("invalid username") // don't echo password attempt to logs
		}

		if portalDoc.Portal.SSH == nil ||
			connmetadata.User() != strings.SplitN(portalDoc.Portal.Username, "@", 2)[0] {
			return nil, fmt.Errorf("invalid username")
		}

		return nil, s.dbPortal.Delete(ctx, portalDoc)
	}

	conn1, newchannels1, requests1, err := cryptossh.NewServerConn(c1, config)
	if err != nil {
		var username string
		if connmetadata != nil { // after a password attempt
			username = connmetadata.User()
		}
		s.baseAccessLog.WithFields(logrus.Fields{
			"remote_addr": c1.RemoteAddr().String(),
			"username":    username,
		}).Warn("authentication failed")

		return err
	}

	accessLog := utillog.EnrichWithPath(s.baseAccessLog, portalDoc.Portal.ID)
	accessLog = accessLog.WithFields(logrus.Fields{
		"hostname":    fmt.Sprintf("master-%d", portalDoc.Portal.SSH.Master),
		"remote_addr": c1.RemoteAddr().String(),
		"username":    portalDoc.Portal.Username,
	})

	accessLog.Print("authentication succeeded")

	openShiftDoc, err := s.dbOpenShiftClusters.Get(ctx, strings.ToLower(portalDoc.Portal.ID))
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", openShiftDoc.OpenShiftCluster.Properties.NetworkProfile.PrivateEndpointIP, 2200+portalDoc.Portal.SSH.Master)

	c2, err := s.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	defer c2.Close()

	key, err := x509.ParsePKCS1PrivateKey(openShiftDoc.OpenShiftCluster.Properties.SSHKey)
	if err != nil {
		return err
	}

	signer, err := cryptossh.NewSignerFromSigner(key)
	if err != nil {
		return err
	}

	conn2, newchannels2, requests2, err := cryptossh.NewClientConn(c2, "", &cryptossh.ClientConfig{
		User: "core",
		Auth: []cryptossh.AuthMethod{
			cryptossh.PublicKeys(signer),
		},
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return err
	}

	t := time.Now()
	accessLog.Print("connected")
	defer func() {
		accessLog.WithFields(logrus.Fields{
			"duration": time.Since(t).Seconds(),
		}).Print("disconnected")
	}()

	keyring := agent.NewKeyring()
	err = keyring.Add(agent.AddedKey{PrivateKey: key})
	if err != nil {
		return err
	}

	return s.proxyConn(accessLog, keyring, conn1, conn2, newchannels1, newchannels2, requests1, requests2)
}

func (s *ssh) proxyConn(accessLog *logrus.Entry, keyring agent.Agent, conn1, conn2 cryptossh.Conn, newchannels1, newchannels2 <-chan cryptossh.NewChannel, requests1, requests2 <-chan *cryptossh.Request) error {
	timer := time.NewTimer(sshTimeout)
	defer timer.Stop()

	var sessionOpened bool

	for {
		select {
		case <-timer.C:
			return nil

		case nc := <-newchannels1:
			if nc == nil {
				return nil
			}

			// on the first c->s session, advertise agent availability
			var firstSession bool
			if !sessionOpened && nc.ChannelType() == "session" {
				firstSession = true
				sessionOpened = true
			}

			go func() {
				_ = s.newChannel(accessLog, nc, conn1, conn2, firstSession)
			}()

		case nc := <-newchannels2:
			if nc == nil {
				return nil
			}

			// hijack and handle incoming s->c agent requests
			if nc.ChannelType() == "auth-agent@openssh.com" {
				go func() {
					_ = s.handleAgent(accessLog, nc, keyring)
				}()
			} else {
				go func() {
					_ = s.newChannel(accessLog, nc, conn2, conn1, false)
				}()
			}

		case request := <-requests1:
			if request == nil {
				return nil
			}

			_ = s.proxyGlobalRequest(request, conn2)

		case request := <-requests2:
			if request == nil {
				return nil
			}

			_ = s.proxyGlobalRequest(request, conn1)
		}
	}
}

func (s *ssh) handleAgent(accessLog *logrus.Entry, nc cryptossh.NewChannel, keyring agent.Agent) error {
	ch, rs, err := nc.Accept()
	if err != nil {
		return err
	}
	defer ch.Close()

	channelLog := accessLog.WithFields(logrus.Fields{
		"channel": nc.ChannelType(),
	})
	channelLog.Printf("opened")
	defer channelLog.Printf("closed")

	go cryptossh.DiscardRequests(rs)

	return agent.ServeAgent(keyring, ch)
}

func (s *ssh) newChannel(accessLog *logrus.Entry, nc cryptossh.NewChannel, conn1, conn2 cryptossh.Conn, firstSession bool) error {
	defer recover.Panic(s.log)

	ch2, rs2, err := conn2.OpenChannel(nc.ChannelType(), nc.ExtraData())
	if err, ok := err.(*cryptossh.OpenChannelError); ok {
		return nc.Reject(err.Reason, err.Message)
	} else if err != nil {
		return err
	}

	ch1, rs1, err := nc.Accept()
	if err != nil {
		return err
	}

	channelLog := accessLog.WithFields(logrus.Fields{
		"channel": nc.ChannelType(),
	})
	channelLog.Printf("opened")
	defer channelLog.Printf("closed")

	if firstSession {
		_, err = ch2.SendRequest("auth-agent-req@openssh.com", true, nil)
		if err != nil {
			return err
		}
	}

	return s.proxyChannel(ch1, ch2, rs1, rs2)
}

func (s *ssh) proxyGlobalRequest(r *cryptossh.Request, c cryptossh.Conn) error {
	ok, payload, err := c.SendRequest(r.Type, r.WantReply, r.Payload)
	if err != nil {
		return err
	}

	return r.Reply(ok, payload)
}

func (s *ssh) proxyRequest(r *cryptossh.Request, ch cryptossh.Channel) error {
	ok, err := ch.SendRequest(r.Type, r.WantReply, r.Payload)
	if err != nil {
		return err
	}

	return r.Reply(ok, nil)
}

func (s *ssh) proxyChannel(ch1, ch2 cryptossh.Channel, rs1, rs2 <-chan *cryptossh.Request) error {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer recover.Panic(s.log)

		defer wg.Done()
		_, _ = io.Copy(ch1, ch2)
		_ = ch1.CloseWrite()
	}()

	go func() {
		defer recover.Panic(s.log)

		defer wg.Done()
		_, _ = io.Copy(ch2, ch1)
		_ = ch2.CloseWrite()
	}()

	go func() {
		defer recover.Panic(s.log)

		defer wg.Done()
		for r := range rs1 {
			err := s.proxyRequest(r, ch2)
			if err != nil {
				break
			}
		}
		_ = ch2.Close()
	}()

	go func() {
		defer recover.Panic(s.log)

		defer wg.Done()
		for r := range rs2 {
			err := s.proxyRequest(r, ch1)
			if err != nil {
				break
			}
		}
		_ = ch1.Close()
	}()

	wg.Wait()
	return nil
}
