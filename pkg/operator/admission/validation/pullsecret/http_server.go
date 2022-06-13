package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

//go:embed manifests
var assets embed.FS

func Deploy(ctx context.Context, cluster *v1alpha1.Cluster, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) {

	dep := deployer.NewDeployer(kubernetescli, dh, assets, "manifests")
	dep.CreateOrUpdate(ctx, cluster, nil)
}

func StartValidator() {
	c := ociRegClient{
		httpClient: &http.Client{Timeout: time.Second * 3},
		ctx:        context.Background(),
		required:   map[string]bool{"quay.io": true},
	}
	http.HandleFunc("/", c.handleRequest)
	log.Fatal(http.ListenAndServeTLS(":8080", "example.crt", "example.key", nil))
}

func (client ociRegClient) handleRequest(w http.ResponseWriter, req *http.Request) {
	review, err := unmarshalReview(req.Body)
	var responseReview admissionv1.AdmissionReview
	encoder := json.NewEncoder(w)

	defer encoder.Encode(responseReview)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("no review in request"))
		return
		//		responseReview = createResponse(review.Request.UID, false, 400, "could not unmarshall the review")
	}
	err = client.validateSecret(review)
	if err != nil {
		responseReview = createResponse(review.Request.UID, false, 400, "some credentials in the pull secret were not valid")
		return
	}
	responseReview = createSuccessResponse(review.Request.UID)
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
