package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"time"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

//go:embed manifests
var assets embed.FS

func Deploy(ctx context.Context, arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) error {
	aroCluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dep := deployer.NewDeployer(kubernetescli, dh, assets, "manifests")
	return dep.CreateOrUpdate(context.Background(), aroCluster, nil)
}

func StartValidator(ctx context.Context, log *logrus.Entry, certPath, keyPath string) error {
	c := ociRegClient{
		httpClient: &http.Client{Timeout: time.Second * 3},
		ctx:        ctx,
		required:   map[string]bool{"quay.io": true},
		log:        log,
	}
	http.HandleFunc("/", c.handleRequest)
	return http.ListenAndServeTLS(":8080", certPath, keyPath, nil)
}

func (client ociRegClient) handleRequest(w http.ResponseWriter, req *http.Request) {
	review, err := unmarshalReview(req.Body)
	encoder := json.NewEncoder(w)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("no review in request"))
		client.log.Println("no review in request")
		return
	}
	err = client.validateSecret(client.log, review)
	if err != nil {
		client.log.Printf("rejected pullsecret for id %s. err: %s", review.Request.UID, err.Error())
		encoder.Encode(createResponse(review.Request.UID, false, 400, "the pullsecret could not be validated"))
		return
	}
	encoder.Encode(createSuccessResponse(review.Request.UID))
}

// most fields are used for failure. For success, prefer createSuccessResponse
func createResponse(uid types.UID, success bool, code int32, message string) admissionv1.AdmissionReview {
	response := admissionv1.AdmissionResponse{}
	response.UID = uid
	if !success {
		response.Allowed = false
		response.Result = &metav1.Status{
			Code:    code,
			Message: message,
		}
	} else {
		response.Allowed = true
	}
	return admissionv1.AdmissionReview{TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}, Response: &response}
}

func createSuccessResponse(uid types.UID) admissionv1.AdmissionReview {
	return createResponse(uid, true, 200, "")
}
