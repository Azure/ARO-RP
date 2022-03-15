package proxy

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyRequestValidationMethod(t *testing.T) {
	server := Server{Subnet: "127.0.0.1/24"}
	_, subnet, err := net.ParseCIDR(server.Subnet)
	if err != nil {
		t.FailNow()
	}
	server.subnet = subnet

	//This should fail because the method is not CONNECT
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://127.0.0.1:123", nil)

	server.validateProxyResquest(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusMethodNotAllowed, response.StatusCode)
		t.FailNow()
	}

	//This should succeed because the method is CONNECT
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodConnect, "127.0.0.1:123", nil)

	server.validateProxyResquest(recorder, request)

	response = recorder.Result()

	if response.StatusCode != http.StatusOK {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusOK, response.StatusCode)
		t.FailNow()
	}

}

func TestProxyRequestValidationHostname(t *testing.T) {

	server := Server{Subnet: "127.0.0.1/24"}
	_, subnet, err := net.ParseCIDR(server.Subnet)
	if err != nil {
		t.FailNow()
	}
	server.subnet = subnet

	//This should fail because the hostname in not valid
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodConnect, "", nil)

	server.validateProxyResquest(recorder, request)

	response := recorder.Result()

	if response.StatusCode != http.StatusBadRequest {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusBadRequest, response.StatusCode)
		t.FailNow()
	}

	//This should succeed because the hostname is valid
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodConnect, "127.0.0.1:8443", nil)

	server.validateProxyResquest(recorder, request)

	response = recorder.Result()

	if response.StatusCode != http.StatusOK {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusOK, response.StatusCode)
		t.FailNow()
	}

}

func TestProxyRequestValidationSubnet(t *testing.T) {

	server := Server{Subnet: "127.0.0.1/24"}
	_, subnet, err := net.ParseCIDR(server.Subnet)
	if err != nil {
		t.FailNow()
	}
	server.subnet = subnet

	//This should succeed because it is in the subnet
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodConnect, "127.0.0.1:1234", nil)

	server.validateProxyResquest(recorder, request)

	response := recorder.Result()

	if response.StatusCode != http.StatusOK {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusOK, response.StatusCode)
		t.FailNow()
	}

	//This should fail because it is not in the subnet
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodConnect, "10.0.0.1:1234", nil)

	server.validateProxyResquest(recorder, request)

	response = recorder.Result()

	if response.StatusCode != http.StatusForbidden {
		t.Logf("Test failed. Reason: was expecting status code to be %d but it was %d", http.StatusForbidden, response.StatusCode)
		t.FailNow()
	}

}
