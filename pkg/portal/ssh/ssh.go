package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	sshNewTimeout = time.Minute
)

type SSH struct {
	env           env.Core
	log           *logrus.Entry
	baseAccessLog *logrus.Entry
	l             net.Listener

	elevatedGroupIDs []string

	dbOpenShiftClusters database.OpenShiftClusters
	dbPortal            database.Portal

	dialer proxy.Dialer

	baseServerConfig *cryptossh.ServerConfig

	hostPubKey cryptossh.PublicKey
}

func New(env env.Core,
	log *logrus.Entry,
	baseAccessLog *logrus.Entry,
	l net.Listener,
	hostKey *rsa.PrivateKey,
	elevatedGroupIDs []string,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbPortal database.Portal,
	dialer proxy.Dialer,
) (*SSH, error) {
	hostPubKey, err := cryptossh.NewPublicKey(&hostKey.PublicKey)
	if err != nil {
		return nil, err
	}

	s := &SSH{
		env:           env,
		log:           log,
		baseAccessLog: baseAccessLog,
		l:             l,

		elevatedGroupIDs: elevatedGroupIDs,

		dbOpenShiftClusters: dbOpenShiftClusters,
		dbPortal:            dbPortal,

		dialer: dialer,

		baseServerConfig: &cryptossh.ServerConfig{
			Config: cryptossh.Config{
				// Per security baseline requirements,
				// https://learn.microsoft.com/en-us/azure/governance/policy/samples/guest-configuration-baseline-linux
				Ciphers:      sshCiphers(),
				KeyExchanges: sshKexAlgorithms(),
				MACs:         sshMACs(),
			},
			PublicKeyAuthAlgorithms: sshPublicKeyAlgorithms(),
		},
		hostPubKey: hostPubKey,
	}

	signer, err := cryptossh.NewSignerFromSigner(hostKey)
	if err != nil {
		return nil, err
	}

	s.baseServerConfig.AddHostKey(signer)

	return s, nil
}

type request struct {
	Master int `json:"master,omitempty"`
}

type response struct {
	Command  string `json:"command,omitempty"`
	Password string `json:"password,omitempty"`
	Error    string `json:"error,omitempty"`
}

// New creates a new temporary password from the request params and sends it
// through the writer
func (s *SSH) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 9 {
		http.Error(w, "invalid resourceId", http.StatusBadRequest)
		return
	}

	resourceID := strings.Join(parts[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		http.Error(w, fmt.Sprintf("invalid resourceId %q", resourceID), http.StatusBadRequest)
		return
	}

	mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediatype != "application/json" {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var req *request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Master < 0 || req.Master > 2 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	elevated := len(stringutils.GroupsIntersect(s.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0
	if !elevated {
		s.sendResponse(w, "", "", "", "Elevated access is required.", s.env.IsLocalDevelopmentMode())
		return
	}

	username := r.Context().Value(middleware.ContextKeyUsername).(string)
	username = strings.SplitN(username, "@", 2)[0]

	password := s.dbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  password,
		TTL: int(sshNewTimeout / time.Second),
		Portal: &api.Portal{
			Username: ctx.Value(middleware.ContextKeyUsername).(string),
			ID:       resourceID,
			SSH: &api.SSH{
				Master: req.Master,
			},
		},
	}

	_, err = s.dbPortal.Create(ctx, portalDoc)
	if err != nil {
		s.internalServerError(w, err)
		return
	}

	host := r.Host
	if strings.ContainsRune(r.Host, ':') {
		host, _, err = net.SplitHostPort(r.Host)
		if err != nil {
			s.internalServerError(w, err)
			return
		}
	}

	s.sendResponse(w, host, username, password, "", s.env.IsLocalDevelopmentMode())
}

func (s *SSH) sendResponse(w http.ResponseWriter, hostname, username, password, error string, isLocalDevelopmentMode bool) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")

	if error != "" {
		err := enc.Encode(response{Error: error})
		if err != nil {
			s.internalServerError(w, err)
		}
		return
	}
	command, err := createLoginCommand(isLocalDevelopmentMode, username, hostname, s.hostPubKey)
	resp := response{Command: command, Password: password}
	if err != nil {
		s.internalServerError(w, err)
	}
	err = enc.Encode(resp)
	if err != nil {
		s.internalServerError(w, err)
	}
}

func (s *SSH) internalServerError(w http.ResponseWriter, err error) {
	s.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// as of 30 Jun 2025 / go 1.24.4 / PR , this server supports the following algorithms
//
// $ nmap --script ssh2-enum-algos localhost -p 2222
// Starting Nmap 7.92 ( https://nmap.org ) at 2025-08-21 09:08 PDT
// Nmap scan report for localhost (127.0.0.1)
// Host is up (0.00020s latency).
// Other addresses for localhost (not scanned): ::1

// PORT     STATE SERVICE
// 2222/tcp open  EtherNetIP-1
// | ssh2-enum-algos:
// |   kex_algorithms: (6)
// |       mlkem768x25519-sha256
// |       ecdh-sha2-nistp256
// |       ecdh-sha2-nistp384
// |       ecdh-sha2-nistp521
// |       diffie-hellman-group14-sha256
// |       kex-strict-s-v00@openssh.com
// |   server_host_key_algorithms: (3)
// |       rsa-sha2-256
// |       rsa-sha2-512
// |       ssh-rsa
// |   encryption_algorithms: (3)
// |       aes256-ctr
// |       aes192-ctr
// |       aes128-ctr
// |   mac_algorithms: (5)
// |       hmac-sha2-256-etm@openssh.com
// |       hmac-sha2-512-etm@openssh.com
// |       hmac-sha2-256
// |       hmac-sha2-512
// |       hmac-sha1
// |   compression_algorithms: (1)
// |_      none
//
// To update the selected algorithms, refer to the Azure security baselines, keeping in mind
// any FIPS requirements.
// https://learn.microsoft.com/en-us/azure/governance/policy/samples/guest-configuration-baseline-linux
// and
// https://liquid.microsoft.com/Web/Views/View/873720#Zrex-3A-2F-2Fsecurityconfigbaselines-2FRequirements-2Fbl-2E00250-2F
//   - In section bl.00250: Linux OS, review the attached "Linux OS Baseline" Excel file
const (
	sshCommand = "echo '{{ .KnownHostLine }}' > {{.Hostname}}_known_host ; " +
		"ssh " +
		"-o UserKnownHostsFile={{.Hostname}}_known_host " +
		"-o Ciphers={{ .Ciphers }} " +
		"-o HostKeyAlgorithms={{ .HostKeyAlgorithms }} " +
		"-o KexAlgorithms={{ .KexAlgorithms }} " +
		"-o MACs={{ .MACs }}" +
		"{{if .IsLocalDevelopmentMode}} -p 2222{{end}} {{.User}}@{{.Hostname}}"
)

// These lists are intentionally not leveraging the cryptossh package's
// constants, as those may change in the future, and they may not be FIPS compliant.

func sshKexAlgorithms() []string {
	return []string{
		cryptossh.KeyExchangeECDHP256,
		cryptossh.KeyExchangeECDHP384,
		cryptossh.KeyExchangeECDHP521,
		cryptossh.KeyExchangeDH14SHA256,
	}
}

func sshHostKeyAlgorithms() []string {
	return []string{
		cryptossh.KeyAlgoRSASHA512,
		cryptossh.KeyAlgoRSASHA256,
		cryptossh.KeyAlgoRSA,
	}
}

func sshCiphers() []string {
	return []string{
		cryptossh.CipherAES256CTR,
		cryptossh.CipherAES192CTR,
		cryptossh.CipherAES128CTR,
	}
}

func sshMACs() []string {
	return []string{
		cryptossh.HMACSHA256ETM,
		cryptossh.HMACSHA512ETM,
		cryptossh.HMACSHA256,
		cryptossh.HMACSHA512,
		cryptossh.HMACSHA1,
	}
}

func sshPublicKeyAlgorithms() []string {
	return []string{
		cryptossh.KeyAlgoED25519,
		cryptossh.KeyAlgoSKED25519,
		cryptossh.KeyAlgoSKECDSA256,
		cryptossh.KeyAlgoECDSA256,
		cryptossh.KeyAlgoECDSA384,
		cryptossh.KeyAlgoECDSA521,
		cryptossh.KeyAlgoRSASHA256,
		cryptossh.KeyAlgoRSASHA512,
		cryptossh.KeyAlgoRSA,
	}
}

func createLoginCommand(isLocalDevelopmentMode bool, user, host string, publicKey cryptossh.PublicKey) (string, error) {
	line := knownhosts.Line([]string{host}, publicKey)
	tmp := template.New("command")
	tmp, err := tmp.Parse(sshCommand)
	if err != nil {
		return "", err
	}
	type fields struct {
		User                   string
		Hostname               string
		KnownHostLine          string
		IsLocalDevelopmentMode bool
		Ciphers                string
		HostKeyAlgorithms      string
		KexAlgorithms          string
		MACs                   string
	}
	var buff bytes.Buffer
	err = tmp.Execute(&buff, fields{
		user,
		host,
		line,
		isLocalDevelopmentMode,
		// Assume we want to use the first supported algorithm
		sshCiphers()[0],
		sshHostKeyAlgorithms()[0],
		sshKexAlgorithms()[0],
		sshMACs()[0],
	})
	return buff.String(), err
}
