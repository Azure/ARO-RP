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
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sync/errgroup"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	keepAliveInterval = time.Second * 30
	keepAliveRequest  = "keep-alive"
)

// This file handles smart proxying of SSH connections between SRE->portal and
// portal->cluster.  We don't want to give the SRE the cluster SSH key, thus
// this has to be an application-level proxy so we can replace the validated
// one-time password that the SRE uses to authenticate with the cluster SSH key.
//
// Given that we're now an application-level proxy, we pull a second trick as
// well: we inject SSH agent forwarding into the portal->cluster connection leg,
// enabling an SRE to ssh from a master node to a worker node without needing an
// additional credential.
//
// SSH itself is a multiplexed protocol.  Within a single TCP connection there
// can exist multiple SSH channels.  Administrative requests and responses can
// also be sent, both on any channel and/or globally.  Channel creations and
// requests can be initiated by either side of the connection.
//
// The golang.org/x/crypto/ssh library exposes the above at a connection level
// as as Conn, chan NewChannel and chan *Request.  All of these have to be
// serviced to prevent the connection from blocking.  Requests to open new
// channels appear on chan NewChannel; global administrative requests appear on
// chan *Request.  Once a new channel is open, a Channel (effectively an
// io.ReadWriteCloser) must be handled plus a further chan *Request for
// channel-scoped administrative requests.
//
// The top half of this file deals with connection instantiation; the bottom
// half deals with proxying Channels and *Requests.

const (
	sshTimeout = time.Hour // never allow a connection to live longer than an hour.
)

func (s *SSH) Run() error {
	go func() {
		defer recover.Panic(s.log)

		for {
			clientConn, err := s.l.Accept()
			if err != nil {
				return
			}

			go func() {
				defer recover.Panic(s.log)

				_ = s.newConn(context.Background(), clientConn)
			}()
		}
	}()

	return nil
}

func (s *SSH) newConn(ctx context.Context, clientConn net.Conn) error {
	defer clientConn.Close()

	config := &cryptossh.ServerConfig{}
	*config = *s.baseServerConfig

	var portalDoc *api.PortalDocument
	var connmetadata cryptossh.ConnMetadata

	// PasswordCallback is called via NewServerConn to validate the one-time
	// password provided.
	config.PasswordCallback = func(_connmetadata cryptossh.ConnMetadata, pw []byte) (*cryptossh.Permissions, error) {
		connmetadata = _connmetadata

		password, err := uuid.FromString(string(pw))
		if err != nil {
			return nil, fmt.Errorf("invalid username") // don't echo password attempt to logs
		}

		portalDoc, err = s.dbPortal.Patch(ctx, password.String(), func(portalDoc *api.PortalDocument) error {
			if portalDoc.Portal.SSH == nil ||
				connmetadata.User() != strings.SplitN(portalDoc.Portal.Username, "@", 2)[0] ||
				portalDoc.Portal.SSH.Authenticated {
				return fmt.Errorf("invalid username")
			}

			portalDoc.Portal.SSH.Authenticated = true

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("invalid username") // don't echo password attempt to logs
		}

		return nil, nil
	}

	// Serve the incoming (SRE->portal) connection.
	upstreamConn, upstreamNewChannels, upstreamRequests, err := cryptossh.NewServerConn(clientConn, config)
	if err != nil {
		if connmetadata != nil { // after a password attempt
			s.baseAccessLog.WithFields(logrus.Fields{
				"remote_addr": clientConn.RemoteAddr().String(),
				"username":    connmetadata.User(),
			}).Warn("authentication failed")
		}

		return err
	}

	// Log the incoming connection attempt.
	accessLog := utillog.EnrichWithPath(s.baseAccessLog, portalDoc.Portal.ID)
	accessLog = accessLog.WithFields(logrus.Fields{
		"hostname":    fmt.Sprintf("master-%d", portalDoc.Portal.SSH.Master),
		"remote_addr": clientConn.RemoteAddr().String(),
		"username":    portalDoc.Portal.Username,
	})

	accessLog.Print("authentication succeeded")

	openShiftDoc, err := s.dbOpenShiftClusters.Get(ctx, strings.ToLower(portalDoc.Portal.ID))
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", openShiftDoc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP, 2200+portalDoc.Portal.SSH.Master)

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

	// Connect the second connection leg (portal->cluster).
	downstreamConn, downstreamNewChannels, downstreamRequests, err := cryptossh.NewClientConn(c2, "", &cryptossh.ClientConfig{
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

	// Proxy channels and requests between the two connections.
	return s.proxyConn(ctx, accessLog, keyring, upstreamConn, downstreamConn, upstreamNewChannels, downstreamNewChannels, upstreamRequests, downstreamRequests)
}

// proxyConn handles incoming new channel and administrative requests.  It calls
// newChannel to handle new channels, each on a new goroutine.
func (s *SSH) proxyConn(ctx context.Context, accessLog *logrus.Entry, keyring agent.Agent, upstreamConn, downstreamConn cryptossh.Conn, upstreamNewChannels, downstreamNewChannels <-chan cryptossh.NewChannel, upstreamRequests, downstreamRequests <-chan *cryptossh.Request) error {
	timer := time.NewTimer(sshTimeout)
	defer timer.Stop()

	var sessionOpened bool

	for {
		select {
		case <-timer.C:
			return nil

		case nc := <-upstreamNewChannels:
			if nc == nil {
				return nil
			}

			// on the first SRE->cluster session, inject an advertisement of
			// agent availability.
			var firstSession bool
			if !sessionOpened && nc.ChannelType() == "session" {
				firstSession = true
				sessionOpened = true
			}

			go func() {
				_ = s.newChannel(ctx, accessLog, nc, upstreamConn, downstreamConn, firstSession)
			}()

		case nc := <-downstreamNewChannels:
			if nc == nil {
				return nil
			}

			if nc.ChannelType() == "auth-agent@openssh.com" {
				// hijack and handle incoming cluster->SRE agent requests
				go func() {
					_ = s.handleAgent(accessLog, nc, keyring)
				}()
			} else {
				go func() {
					_ = s.newChannel(ctx, accessLog, nc, downstreamConn, upstreamConn, false)
				}()
			}

		case request := <-upstreamRequests:
			if request == nil {
				return nil
			}

			_ = s.proxyGlobalRequest(request, downstreamConn)

		case request := <-downstreamRequests:
			if request == nil {
				return nil
			}

			_ = s.proxyGlobalRequest(request, upstreamConn)
		}
	}
}

func (s *SSH) handleAgent(accessLog *logrus.Entry, nc cryptossh.NewChannel, keyring agent.Agent) error {
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

// newChannel handles an incoming request to create a new channel.  If the
// channel creation is successful, it calls proxyChannel to proxy the channel
// between SRE and cluster.
func (s *SSH) newChannel(ctx context.Context, accessLog *logrus.Entry, nc cryptossh.NewChannel, upstreamConn, downstreamConn cryptossh.Conn, firstSession bool) error {
	defer recover.Panic(s.log)

	ch2, rs2, err := downstreamConn.OpenChannel(nc.ChannelType(), nc.ExtraData())
	if errAsOpenChannel, ok := err.(*cryptossh.OpenChannelError); ok {
		return nc.Reject(errAsOpenChannel.Reason, errAsOpenChannel.Message)
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

		go s.keepAliveConn(ctx, ch1)
	}

	return s.proxyChannel(ch1, ch2, rs1, rs2)
}

func (s *SSH) proxyGlobalRequest(r *cryptossh.Request, c cryptossh.Conn) error {
	ok, payload, err := c.SendRequest(r.Type, r.WantReply, r.Payload)
	if err != nil {
		return err
	}

	return r.Reply(ok, payload)
}

func (s *SSH) proxyRequest(r *cryptossh.Request, ch cryptossh.Channel) error {
	ok, err := ch.SendRequest(r.Type, r.WantReply, r.Payload)
	if err != nil {
		return err
	}

	return r.Reply(ok, nil)
}

func (s *SSH) proxyChannel(ch1, ch2 cryptossh.Channel, rs1, rs2 <-chan *cryptossh.Request) error {
	g := errgroup.Group{}

	g.Go(func() error {
		defer recover.Panic(s.log)
		defer func() {
			_ = ch1.CloseWrite()
		}()
		_, err := io.Copy(ch1, ch2)
		if err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		defer recover.Panic(s.log)
		defer func() {
			_ = ch2.CloseWrite()
		}()
		_, err := io.Copy(ch2, ch1)
		if err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		defer recover.Panic(s.log)

		for r := range rs1 {
			err := s.proxyRequest(r, ch2)
			if err != nil {
				break
			}
		}
		return ch2.Close()
	})

	g.Go(func() error {
		defer recover.Panic(s.log)

		for r := range rs2 {
			err := s.proxyRequest(r, ch1)
			if err != nil {
				break
			}
		}
		return ch1.Close()
	})

	return g.Wait()
}

func (s *SSH) keepAliveConn(ctx context.Context, channel cryptossh.Channel) {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := channel.SendRequest(keepAliveRequest, true, nil)
			if err != nil {
				s.log.Debugf("connection failed keep-alive check, closing it. Error: %s", err)
				// Connection is gone
				channel.Close()
				return
			}
		}
	}
}
