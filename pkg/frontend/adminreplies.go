package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
)

type StreamResponder interface {
	ReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error)
	AdminReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error)
}

type defaultResponder struct {
}

func (d defaultResponder) ReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error) {
	for k, v := range header {
		w.Header()[k] = v
	}

	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			log.Info(err)
			api.WriteCloudError(w, err)
			return
		case statusCodeError:
			w.WriteHeader(int(err))
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	}

	io.Copy(w, reader)
	_, _ = w.Write([]byte{'\n'})
}

func (d defaultResponder) AdminReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error) {
	if apiErr, ok := err.(kerrors.APIStatus); ok {
		status := apiErr.Status()

		var target string
		if status.Details != nil {
			gk := schema.GroupKind{
				Group: status.Details.Group,
				Kind:  status.Details.Kind,
			}

			target = fmt.Sprintf("%s/%s", gk, status.Details.Name)
		}

		err = &api.CloudError{
			StatusCode: int(status.Code),
			CloudErrorBody: &api.CloudErrorBody{
				Code:    string(status.Reason),
				Message: status.Message,
				Target:  target,
			},
		}
	}

	d.ReplyStream(log, w, header, reader, err)
}
