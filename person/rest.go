package person

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/golang/glog"
)

const DEFAULT_API_TIMEOUT = time.Duration(60 * time.Second)

type PersonResult struct {
	// Code should match the HTTP status code.
	Code int `json:"code"`

	// Message should contain a short description of the result of the requested
	// operation.
	Message *string `json:"message"`
}

type PersonError struct {
	PersonResult

	// verb is the HTTP verb (GET, POST, PUT, PATCH, or DELETE) that was
	// used in the request that resulted in the error.
	Verb string

	// url is the URL that was used in the request that resulted in the error.
	Url string

	// HttpStatusCode is the HTTP response status code (e.g., 200, 404, etc.).
	HttpStatusCode int

	// err contains a descriptive error object for error cases other than HTTP
	// errors (i.e., non-2xx responses), such as socket errors or malformed JSON.
	err error
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type PersonSession struct {
	host string

	insecure bool

	prefix string

	// timeout specifies time limit for API request. Default value set to 60 seconds
	timeout time.Duration

	// internal: re-usable transport to enable connection reuse
	transport *http.Transport

	// internal: reusable client
	client HttpClient
}

// Error implements the error interface.
func (err PersonError) Error() string {
	var msg string

	if err.err != nil {
		msg = fmt.Sprintf("error: %v", err.err)
	} else if err.Message != nil {
		msg = fmt.Sprintf("HTTP code: %d; error from Person: %s",
			err.HttpStatusCode, *err.Message)
	} else {
		msg = fmt.Sprintf("HTTP code: %d.", err.HttpStatusCode)
	}

	return fmt.Sprintf("Encountered an error on %s request to URL %s: %s",
		err.Verb, err.Url, msg)
}

func (personsess *PersonSession) newPersonRequest(verb string, url string, payload io.Reader) (*http.Request, PersonError) {
	req, err := http.NewRequest(verb, url, payload)
	errorResult := PersonError{Verb: verb, Url: url}
	if err != nil {
		errorResult.err = fmt.Errorf("http.NewRequest failed: %v", err)
		return nil, errorResult
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Accept", "application/json")
	return req, errorResult
}

// Returns http.Response for accessing the whole http Response struct including headers and response body
func (personsess *PersonSession) RestRequest(verb string, uri string, payload interface{}) (*http.Response, error) {
	return personsess.restRequest(verb, uri, payload)
}

// Returns http.Response if successful
// Note: The caller of the function is responsible for doing resp.Body.Close()
func (personsess *PersonSession) restRequest(verb string, uri string, payload interface{}) (*http.Response, error) {
	url := personsess.prefix + uri
	var payloadIO io.Reader
	if payload != nil {
		jsonStr, err := json.Marshal(payload)
		if err != nil {
			return nil, PersonError{Verb: verb, Url: url, err: err}
		}
		payloadIO = bytes.NewBuffer(jsonStr)
	}

	req, errorResult := personsess.newPersonRequest(verb, url, payloadIO)
	if errorResult.err != nil {
		return nil, errorResult
	}

	log.Printf("Request %s with url %s", verb, url)
	resp, err := personsess.client.Do(req)
	if err != nil {
		glog.Errorf("Client error for URI: %+v. Error: %+v", url, err.Error())
		dump, dumpErr := httputil.DumpRequestOut(req, true)
		if dumpErr != nil {
			glog.Error("Error while dumping request. Still retrying.")
		}
		debug(dump, dumpErr)
	}

	if resp.StatusCode > 299 {
		resp.Body.Close()
		fmt.Printf("Error in url %s; Status Code %d", url, resp.StatusCode)
		return nil, errors.New("Rest request error, returning to caller")
	}
	return resp, nil
}

func NewPersonSession(host string, options ...func(*PersonSession) error) (*PersonSession, error) {
	if flag.Parsed() == false {
		flag.Parse()
	}
	personsess := &PersonSession{
		host: host,
	}

	personsess.prefix = "http://" + personsess.host + "/"
	personsess.insecure = false

	// set default timeout
	if personsess.timeout == 0 {
		personsess.timeout = DEFAULT_API_TIMEOUT
	}

	if personsess.client == nil {
		// create default transport object
		if personsess.transport == nil {
			personsess.transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}

		// attach transport object to client
		personsess.client = &http.Client{
			Transport: personsess.transport,
			Timeout:   personsess.timeout,
		}
	}
	return personsess, nil
}

// fetchBody fetches the response body from the http.Response returned from restRequest
func (personsess *PersonSession) fetchBody(verb, url string, resp *http.Response) (result []byte, err error) {
	errorResult := PersonError{HttpStatusCode: resp.StatusCode, Verb: verb, Url: url}

	if resp.StatusCode == 204 {
		// no content in the response
		return result, nil
	}
	// It cannot be assumed that the error will always be from server side in response.
	// Error could be from HTTP client side which will not have body in response.
	// Need to change our API resp handling design if we want to handle client side errors separately.

	// Below block will take care for errors without body.
	if resp.Body == nil {
		glog.Errorf("Encountered client side error: %+v", resp)
		errorResult.Message = &resp.Status
		return result, errorResult
	}

	defer resp.Body.Close()
	result, err = ioutil.ReadAll(resp.Body)
	if err == nil {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			mres, merr := personsess.convertPersonResponseToMapInterface(result)
			glog.Infof("Error code %v parsed resp: %v err %v",
				resp.StatusCode, mres, merr)
			emsg := fmt.Sprintf("%v", mres)
			errorResult.Message = &emsg
		} else {
			return result, nil
		}
	} else {
		errmsg := fmt.Sprintf("Response body read failed: %v", err)
		errorResult.Message = &errmsg
		glog.Errorf("Error in reading uri %v %v", url, err)
	}
	return result, errorResult
}

func (personsess *PersonSession) restRequestInterfaceResponse(verb string, url string,
	payload interface{}, response interface{}) error {
	httpResponse, rerror := personsess.restRequest(verb, url, payload)
	if rerror != nil {
		return rerror
	}
	var res []byte
	var err error
	if res, err = personsess.fetchBody(verb, url, httpResponse); err != nil {
		return err
	}

	if len(res) == 0 {
		return nil
	} else {
		json.Unmarshal(res, &response)
	}
	return nil
}

func (personsess *PersonSession) convertPersonResponseToMapInterface(resbytes []byte) (interface{}, error) {
	var result interface{}
	err := json.Unmarshal(resbytes, &result)
	return result, err
}

func debug(data []byte, err error) {
	if err == nil {
		glog.Infof("%s\n\n", data)
	} else {
		glog.Errorf("%s\n\n", err)
	}
}

// Get issues a GET request against the Personsess REST API.
func (personsess *PersonSession) Get(uri string, response interface{}) error {
	return personsess.restRequestInterfaceResponse("GET", uri, nil, response)
}

// Post issues a POST request against the Personsess REST API.
func (personsess *PersonSession) Post(uri string, payload interface{}, response interface{}) error {
	return personsess.restRequestInterfaceResponse("POST", uri, payload, response)
}

// Put issues a PUT request against the Personsess REST API.
func (personsess *PersonSession) Put(uri string, payload interface{}, response interface{}) error {
	return personsess.restRequestInterfaceResponse("PUT", uri, payload, response)
}

// Post issues a PATCH request against the Personsess REST API.
// allowed patchOp - add, replace, remove
func (personsess *PersonSession) Patch(uri string, payload interface{}, patchOp string, response interface{}) error {
	var patchPayload = make(map[string]interface{})
	patchPayload[patchOp] = payload
	glog.Infof(" PATCH OP %v data %v", patchOp, payload)
	return personsess.restRequestInterfaceResponse("PATCH", uri, patchPayload, response)
}

// Delete issues a DELETE request against the Personsess REST API.
func (personsess *PersonSession) Delete(uri string, params ...interface{}) error {
	var payload, response interface{}
	if len(params) > 0 {
		payload = params[0]
		if len(params) == 2 {
			response = params[1]
		}
	}
	return personsess.restRequestInterfaceResponse("DELETE", uri, payload, response)
}
