package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilcert "github.com/Azure/ARO-RP/pkg/util/cert"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type etcdrenew struct {
	log           *logrus.Entry
	k             adminactions.KubeActions
	doc           *api.OpenShiftClusterDocument
	secretNames   []string
	backupSecrets map[string][]byte
	lastRevision  int32
	timeout       time.Duration
}

var etcdOperatorControllerConditionsExpected = map[string]operatorv1.ConditionStatus{
	"EtcdCertSignerControllerDegraded": operatorv1.ConditionFalse,
}

var etcdOperatorConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
}

func (f *frontend) postAdminOpenShiftClusterEtcdCertificateRenew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	err := f._postAdminOpenShiftClusterEtcdCertificateRenew(ctx, resourceID, log, 30*time.Minute)

	adminReply(log, w, nil, nil, err)
}

// validate cluster is <4.9 and etcd operator is in expected state
// Secrets exists, unexpired and close to expiry
// backup and delete secrets, if backupAndDelete is set True
func (e *etcdrenew) validateEtcdAndBackupDeleteSecretOnFlagSet(ctx context.Context, backupAndDelete bool) error {
	s := []steps.Step{
		steps.Action(e.validateEtcdOperatorControllersState),
		steps.Action(e.validateEtcdOperatorState),
		steps.Action(e.validateEtcdCertsExistsAndExpiry),
	}

	if backupAndDelete {
		s = append(s,
			steps.Action(e.fetchEtcdCurrentRevision),
			steps.Action(e.backupEtcdSecrets),
			steps.Action(e.deleteEtcdSecrets),
		)
	}

	_, err := steps.Run(ctx, e.log, 10*time.Second, s, nil)
	if err != nil {
		return err
	}
	return nil
}

// Etcd secrets are deleted or updated, a new revision is will put and applied
// This function polls if a new revision is applied successfully
func (e *etcdrenew) isEtcDRootCertRenewed(ctx context.Context) error {
	s := []steps.Step{
		steps.Condition(e.isEtcdRevised, e.timeout, true),
	}
	_, err := steps.Run(ctx, e.log, 30*time.Second, s, nil)
	if err != nil {
		return err
	}
	return nil
}

func (e *etcdrenew) revertChanges(ctx context.Context) error {
	s := []steps.Step{
		steps.Action(e.fetchEtcdCurrentRevision),
		steps.Action(e.recoverEtcdSecrets),
		steps.Condition(e.isEtcdRevised, 30*time.Minute, true),
	}
	_, err := steps.Run(ctx, e.log, 10*time.Second, s, nil)
	if err != nil {
		return err
	}
	return nil
}

// runs the etcd renewal and recovery
func (e *etcdrenew) run(ctx context.Context) error {
	if err := e.validateClusterVersion(ctx); err != nil {
		return err
	}

	// Fetch secretNames using nodeNames
	for i := 0; i < 3; i++ {
		nodeName := e.doc.OpenShiftCluster.Properties.InfraID + "-master-" + strconv.Itoa(i)
		for _, prefix := range []string{"etcd-peer-", "etcd-serving-", "etcd-serving-metrics-"} {
			e.secretNames = append(e.secretNames, prefix+nodeName)
		}
	}

	// validate etcd and certificates, backup and delete secrets
	if err := e.validateEtcdAndBackupDeleteSecretOnFlagSet(ctx, true); err != nil {
		return err
	}

	// Once secrets are deleted, the operator recreates the secrets and a new etcd revision is applied
	// On failure, proceed for recovery by applying the backupsecrets on the cluster again
	// On success, verify the etcd state, certificates
	err := e.isEtcDRootCertRenewed(ctx)
	if err != nil {
		e.log.Infoln("Attempting to recover from backup, and wait for new revision to be applied after recovery")
		if err = e.revertChanges(ctx); err != nil {
			return err
		}
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "etcd renewal failed, recovery performed to revert the changes.")
	}

	e.log.Infoln("Etcd certificates are renewed and new revision is applied, verifying.")
	err = e.validateEtcdAndBackupDeleteSecretOnFlagSet(ctx, false)
	if err != nil {
		return err
	}

	// validates if the etcd certificates are renewed
	return e.validateEtcdCertsRenewed(ctx)
}

func (f *frontend) _postAdminOpenShiftClusterEtcdCertificateRenew(ctx context.Context, resourceID string, log *logrus.Entry, timeout time.Duration) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}
	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", r.ResourceType, r.ResourceName, r.ResourceGroup)
	case err != nil:
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	e := &etcdrenew{
		log:           log,
		k:             k,
		doc:           doc,
		secretNames:   nil,
		backupSecrets: make(map[string][]byte),
		lastRevision:  0,
		timeout:       timeout,
	}

	if err := e.run(ctx); err != nil {
		log.Errorf("Geneva Action run failed with error %s", err.Error())
		return err
	}

	log.Infoln("Done")
	return nil
}

func (e *etcdrenew) validateClusterVersion(ctx context.Context) error {
	e.log.Infoln("validating cluster version now")
	rawCV, err := e.k.KubeGet(ctx, "ClusterVersion.config.openshift.io", "", "version")
	if err != nil {
		return err
	}
	cv := &configv1.ClusterVersion{}
	err = codec.NewDecoderBytes(rawCV, &codec.JsonHandle{}).Decode(cv)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode clusterversion, %s", err.Error()))
	}
	clusterVersion, err := version.GetClusterVersion(cv)
	if err != nil {
		return err
	}
	// ETCD ceritificates are autorotated by the operator when close to expiry for cluster running 4.9+
	if !clusterVersion.Lt(version.NewVersion(4, 9)) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "etcd certificate renewal is not needed for cluster running version 4.9+")
	}
	e.log.Infof("validated: cluster version is %s", clusterVersion)

	return nil
}

func (e *etcdrenew) validateEtcdOperatorControllersState(ctx context.Context) error {
	e.log.Infoln("validating etcdOperator Controllers state now")
	rawEtcd, err := e.k.KubeGet(ctx, "etcd.operator.openshift.io", "", "cluster")
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	etcd := &operatorv1.Etcd{}
	err = codec.NewDecoderBytes(rawEtcd, &codec.JsonHandle{}).Decode(etcd)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode etcd object, %s", err.Error()))
	}
	for _, c := range etcd.Status.Conditions {
		if _, ok := etcdOperatorControllerConditionsExpected[c.Type]; !ok {
			continue
		}
		if etcdOperatorControllerConditionsExpected[c.Type] != c.Status {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%s is in state %s, quiting.", c.Type, c.Status)
		}
	}
	e.log.Infoln("EtcdOperator Controllers state is validated.")

	return nil
}

func (e *etcdrenew) validateEtcdOperatorState(ctx context.Context) error {
	e.log.Infoln("validating Etcd Operator state")
	rawEtcdOperator, err := e.k.KubeGet(ctx, "ClusterOperator.config.openshift.io", "", "etcd")
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	etcdOperator := &configv1.ClusterOperator{}
	err = codec.NewDecoderBytes(rawEtcdOperator, &codec.JsonHandle{}).Decode(etcdOperator)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode etcd operator, %s", err.Error()))
	}
	for _, c := range etcdOperator.Status.Conditions {
		if _, ok := etcdOperatorConditionsExpected[c.Type]; !ok {
			continue
		}
		if etcdOperatorConditionsExpected[c.Type] != c.Status {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Etcd Operator is not in expected state, quiting.")
		}
		if c.Type == configv1.OperatorAvailable && c.Reason != "AsExpected" {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Etcd Operator Available state is not AsExpected, quiting.")
		}
	}
	e.log.Infoln("Etcd operator state validated.")

	return nil
}

func (e *etcdrenew) validateEtcdCertsExistsAndExpiry(ctx context.Context) error {
	e.log.Infoln("validating if etcd certs exists and expiry")

	for _, secretname := range e.secretNames {
		e.log.Infof("validating secret %s", secretname)
		cert, err := e.k.KubeGet(ctx, "Secret", namespaceEtcds, secretname)
		if err != nil {
			return err
		}

		var u unstructured.Unstructured
		var secret corev1.Secret
		if err = json.Unmarshal(cert, &u); err != nil {
			return err
		}
		err = kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &secret)
		if err != nil {
			return err
		}
		_, certData, err := utilpem.Parse(secret.Data[corev1.TLSCertKey])
		if err != nil {
			return err
		}
		if len(certData) < 1 {
			return fmt.Errorf("invalid cert data when parsing secret: %s", secret.Name)
		}
		if utilcert.IsCertExpired(certData[0]) {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "secret %s is already expired, quitting.", secretname)
		}
	}
	e.log.Infoln("Etcd certs exits, are not expired")

	return nil
}

func (e *etcdrenew) validateEtcdCertsRenewed(ctx context.Context) error {
	e.log.Infoln("validating if etcd certs are renewed")
	isError := false

	for _, secretname := range e.secretNames {
		e.log.Infof("validating secret %s", secretname)
		cert, err := e.k.KubeGet(ctx, "Secret", namespaceEtcds, secretname)
		if err != nil {
			return err
		}

		var u unstructured.Unstructured
		var secret corev1.Secret
		if err = json.Unmarshal(cert, &u); err != nil {
			return err
		}
		err = kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &secret)
		if err != nil {
			return err
		}
		_, certData, err := utilpem.Parse(secret.Data[corev1.TLSCertKey])
		if err != nil {
			return err
		}

		// etcd operator renews certificates for another 3 years, 1000+ days (3*365)
		e.log.Infof("certificate '%s' expiration date is '%s'", secretname, certData[0].NotAfter)
		if utilcert.DaysUntilExpiration(certData[0]) < 1000 {
			isError = true
			e.log.Errorf("certificate %s is not renewed successfully.", secretname)
		}
	}

	if isError {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "etcd certificates renewal not successful, as at least one or all certificates are not renewed")
	}

	e.log.Infoln("etcd certificates are successfully renewed")
	return nil
}

func (e *etcdrenew) fetchEtcdCurrentRevision(ctx context.Context) error {
	e.log.Infoln("fetching etcd Current Revision now")
	rawEtcd, err := e.k.KubeGet(ctx, "etcd.operator.openshift.io", "", "cluster")
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	etcd := &operatorv1.Etcd{}
	err = codec.NewDecoderBytes(rawEtcd, &codec.JsonHandle{}).Decode(etcd)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode etcd object, %s", err.Error()))
	}

	e.lastRevision = etcd.Status.LatestAvailableRevision
	e.log.Infof("Current Etcd Revision is %d", e.lastRevision)

	return nil
}

// backup existing etcd secrets in the cluster, into runtime variable,
func (e *etcdrenew) backupEtcdSecrets(ctx context.Context) error {
	e.log.Infoln("backing up etcd secrets now")
	for _, secretname := range e.secretNames {
		err := retry.OnError(wait.Backoff{
			Steps:    10,
			Duration: 2 * time.Second,
		}, func(err error) bool {
			return errors.IsBadRequest(err) || errors.IsInternalError(err) || errors.IsServerTimeout(err)
		}, func() error {
			e.log.Infof("Backing up secret %s", secretname)
			data, err := e.k.KubeGet(ctx, "Secret", namespaceEtcds, secretname)
			if err != nil {
				return err
			}
			secret := &corev1.Secret{}
			err = codec.NewDecoderBytes(data, &codec.JsonHandle{}).Decode(secret)
			if err != nil {
				return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode secret, %s", err.Error()))
			}
			secret.CreationTimestamp = metav1.Time{
				Time: time.Now(),
			}
			secret.ObjectMeta.ResourceVersion = ""
			secret.ObjectMeta.UID = ""

			var cert []byte
			err = codec.NewEncoderBytes(&cert, &codec.JsonHandle{}).Encode(secret)
			if err != nil {
				return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to encode secret, %s", err.Error()))
			}
			e.backupSecrets[secretname] = cert
			return nil
		})
		if err != nil {
			return err
		}
	}

	e.log.Infoln("backing up etcd secrets done")
	return nil
}

// delete the etcd secrets and on successful deletion,
// valid secrets will be recreated and a new revision will be applied by the etcd operator
func (e *etcdrenew) deleteEtcdSecrets(ctx context.Context) error {
	e.log.Infoln("deleting etcd secrets now")
	for _, secretname := range e.secretNames {
		err := retry.OnError(wait.Backoff{
			Steps:    10,
			Duration: 2 * time.Second,
		}, func(err error) bool {
			return errors.IsBadRequest(err) || errors.IsInternalError(err) || errors.IsServerTimeout(err)
		}, func() error {
			e.log.Infof("Deleting secret %s", secretname)
			err := e.k.KubeDelete(ctx, "Secret", namespaceEtcds, secretname, false, nil)
			if err != nil {
				return err
			}
			e.log.Infof("Secret deleted %s", secretname)
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Checks if the new revision is put on the etcd and validates if all the nodes are running the same revision
func (e *etcdrenew) isEtcdRevised(ctx context.Context) (etcdCheck bool, retry bool, err error) {
	isAtRevision := true
	rawEtcd, err := e.k.KubeGet(ctx, "etcd.operator.openshift.io", "", "cluster")
	if err != nil {
		e.log.Warnf(err.Error())
		return false, true, nil
	}
	etcd := &operatorv1.Etcd{}
	err = codec.NewDecoderBytes(rawEtcd, &codec.JsonHandle{}).Decode(etcd)
	if err != nil {
		e.log.Warnf(err.Error())
		return false, true, nil
	}

	// no new revision is observed.
	if e.lastRevision == etcd.Status.LatestAvailableRevision {
		e.log.Infof("last revision is %d, latest available revision is %d", e.lastRevision, etcd.Status.LatestAvailableRevision)
		return false, true, nil
	}
	for _, s := range etcd.Status.NodeStatuses {
		e.log.Infof("Current Revision for node %s is %d, expected revision is %d", s.NodeName, s.CurrentRevision, etcd.Status.LatestAvailableRevision)
		if s.CurrentRevision != etcd.Status.LatestAvailableRevision {
			isAtRevision = false
			break
		}
	}

	return isAtRevision, true, nil
}

// Applies the backedup etcd secret and applies them on the cluster
func (e *etcdrenew) recoverEtcdSecrets(ctx context.Context) error {
	e.log.Infoln("recovering etcd secrets now")
	for secretname, data := range e.backupSecrets {
		err := retry.OnError(wait.Backoff{
			Steps:    10,
			Duration: 2 * time.Second,
		}, func(err error) bool {
			return errors.IsBadRequest(err) || errors.IsInternalError(err) || errors.IsServerTimeout(err)
		}, func() error {
			// skip secrets which are already recovered
			e.log.Infof("Recovering secret %s", secretname)
			obj := &unstructured.Unstructured{}
			err := obj.UnmarshalJSON(data)
			if err != nil {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
			}
			err = e.k.KubeCreateOrUpdate(ctx, obj)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	e.log.Infoln("recovered etcd secrets")

	return nil
}
