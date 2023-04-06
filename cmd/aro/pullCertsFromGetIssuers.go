package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

func genCrt(name string, base64EncodedCert string, path string) {
	filename := path + "/" + name + ".crt"
	data, err := base64.StdEncoding.DecodeString(base64EncodedCert)

	if err != nil {
		panic(err)
	}

	f, err := os.Create(filename)

	if err != nil {
		panic(err)
	}

	defer f.Close()
	f.Write(data)
}

// https://aka.ms/getissuers
// The v3 endpoint can be used to create certs
// For example https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=ame
// returns the ame certs
func pullCertsFromGetIssuers(url string, outPath string) {
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		err := os.MkdirAll(outPath, 0755)
		if err != nil {
			panic(err)
		}
	}

	response, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	var data map[string]interface{}
	json.Unmarshal(body, &data)
	roots := data["RootsInfos"].([]interface{})
	for _, root := range roots {
		rootName := root.(map[string]interface{})["rootName"].(string)
		caName := root.(map[string]interface{})["CaName"].(string)
		name := caName + "_root_" + path.Base(rootName)
		rootBody := root.(map[string]interface{})["Body"].(string)
		genCrt(name, rootBody, outPath)
		intermediates := root.(map[string]interface{})["Intermediates"].([]interface{})
		for _, intermediate := range intermediates {
			intermediateName := intermediate.(map[string]interface{})["IntermediateName"].(string)
			name = caName + "_intermediate_" + path.Base(intermediateName)
			intermediateBody := intermediate.(map[string]interface{})["Body"].(string)
			genCrt(name, intermediateBody, outPath)
		}
	}
}
