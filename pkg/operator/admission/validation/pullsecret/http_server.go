package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	gofrsuuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const rpImage = "RP_IMAGE"
const azureRegistry = "AZURE_REGISTRY"
const httpTimeout = time.Second * 3

//go:embed manifests
var assets embed.FS

type deploymentData struct {
	Image         string
	AzureRegistry string
}

func Deploy(ctx context.Context, arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) error {
	aroCluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	image, isSet := os.LookupEnv(rpImage)
	if !isSet {
		return fmt.Errorf("%s is not set", rpImage)
	}
	acr, isSet := os.LookupEnv(azureRegistry)
	if !isSet {
		return fmt.Errorf("%s is not set", azureRegistry)
	}

	dep := deployer.NewDeployer(kubernetescli, dh, assets, "manifests")
	return dep.CreateOrUpdate(context.Background(), aroCluster, deploymentData{Image: image, AzureRegistry: acr})
}

func StartValidator(ctx context.Context, log *logrus.Entry, certPath, keyPath, containerRegistryUrl string) error {
	c := ociRegClient{
		httpClient: &http.Client{Timeout: httpTimeout},
		ctx:        ctx,
		required: map[string]bool{
			"quay.io":                     true,
			"registry.connect.redhat.com": true,
			"registry.redhat.io":          true},
		log:           log,
		azureRegistry: containerRegistryUrl,
	}
	http.HandleFunc("/", c.handleRequest)
	return http.ListenAndServeTLS(":8080", certPath, keyPath, nil)
}

const noReviewError = "no review in request"

func (client ociRegClient) handleRequest(w http.ResponseWriter, req *http.Request) {
	review, err := unmarshalReview(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(noReviewError))
		client.log.Println(noReviewError)
		return
	}

	// user input sanitation , codeql and cwe related
	requestUid, err := validateReviewUID(review.Request.UID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	encoder := json.NewEncoder(w)
	err = client.validateSecret(client.log, review)
	if err != nil {
		client.log.Printf("rejected pullsecret for id %s. err: %s", requestUid, err.Error())
		encoder.Encode(
			createResponse(review.Request.UID,
				false,
				http.StatusBadRequest,
				fmt.Sprintf("the pullsecret could not be validated. err: %s", err)))
		return
	}
	encoder.Encode(createSuccessResponse(review.Request.UID))
}

// CWE and github complain about user input sanitation
func validateReviewUID(uid types.UID) (string, error) {
	parsed, err := gofrsuuid.FromString(string(uid))
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

// most fields are used for failure. For success, prefer createSuccessResponse
func createResponse(uid types.UID, success bool, code int32, message string) admissionv1.AdmissionReview {
	response := admissionv1.AdmissionResponse{}
	response.UID = uid
	response.Allowed = true
	if !success {
		response.Allowed = false
		response.Result = &metav1.Status{
			Code:    code,
			Message: message,
		}
	}
	return admissionv1.AdmissionReview{TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}, Response: &response}
}

func createSuccessResponse(uid types.UID) admissionv1.AdmissionReview {
	return createResponse(uid, true, http.StatusOK, "")
}
