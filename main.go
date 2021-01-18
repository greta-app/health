package main

import (
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
	"os"
	"strconv"
	"strings"
	"time"
)

type test struct {
	Url                       string            `json:"url"`
	ExpectedResponseCodeRange string            `json:"code_range"`
	Method                    string            `json:"method"`
	Contains                  string            `json:"contains"`
	Headers                   map[string]string `json:"headers"`
}

var scriptPath string
var isVerbose bool

func main() {
	port := flag.Int("port", 8023, "Port to listen for incoming requests")
	scriptPathPtr := flag.String("scriptPath", "script.json", "Path to testing scripts")
	isVerbosePtr := flag.Bool("verbose", false, "is verbose")
	isHelp := flag.Bool("help", false, "Prints help")

	flag.Parse()

	if *isHelp {
		log.Printf("Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()
		return
	}

	scriptPath = *scriptPathPtr

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Fatalf("file does not exist: %s", scriptPath)
	}

	_, err := parseScript()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handle)
	localAddr := fmt.Sprintf(":%d", *port)
	log.Printf("start process on port %s, for the script path %s \n", localAddr, scriptPath)
	err = http.ListenAndServe(localAddr, nil)
	if err != nil {
		log.Fatal(err)
	}

	isVerbose = *isVerbosePtr
}

func handle(w http.ResponseWriter, req *http.Request) {
	tests, err := parseScript()
	if err != nil {
		handleErr(w, err)
		return
	}

	log.Printf("will execute %d tests\n", len(tests))
	errs := make([]string, 0, len(tests))
	for _, test := range tests {
		start := time.Now()
		err = executeTest(test)
		if err != nil {
			log.Printf("test's failed: %v", err)
		}
		log.Printf("took: %v", time.Since(start))

		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		handleErr(w, errors.New(strings.Join(errs, "\n")))
		return
	}

	handleSuccess(w)
}

func executeTest(t test) error {
	maxRespCode, minRespCode, err := parseExpectedCodesRange(t.ExpectedResponseCodeRange)
	req, err := http.NewRequest(
		t.Method,
		t.Url,
		strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	for key, value := range t.Headers {
		req.Header.Add(key, value)
	}

	connectionTimeout := 10 * time.Second
	transport := &http.Transport{
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		ResponseHeaderTimeout: connectionTimeout,
	}

	client := http.Client{Transport: transport}

	if isVerbose {
		dump, _ := httputil.DumpRequest(req, true)
		log.Printf("Input context: %+v, raw request: %s\n", t, string(dump))
	} else {
		log.Printf("Calling server with input parameters: %+v", t)
	}

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			closeErr := resp.Body.Close()
			if closeErr != nil {
				log.Println(closeErr)
			}
		}
		return fmt.Errorf("request failed with error: %v", err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			log.Println(closeErr)
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err == io.EOF {
		log.Printf("Empty body in the response, status: %d\n", resp.StatusCode)
	}
	if err != nil {
		return fmt.Errorf("reading of the request body failed with error: %v, status: %d", err, resp.StatusCode)
	}

	log.Printf("Got response status code: '%d'\n", resp.StatusCode)

	if resp.StatusCode < minRespCode && resp.StatusCode > maxRespCode {
		return fmt.Errorf("test failed: the response code %d is not in expected range: %s, resp body: %s", resp.StatusCode, t.ExpectedResponseCodeRange, string(respBody))
	}

	if t.Contains == "" {
		log.Printf("test %+v execution success\n", t)
		return nil
	}

	if !strings.Contains(string(respBody), t.Contains) {
		return fmt.Errorf("test failed: could not find expected string '%s' in response body '%s'", t.Contains, string(respBody))
	}

	log.Printf("test %+v execution success\n", t)
	return nil
}

func handleErr(w http.ResponseWriter, err error) {
	w.WriteHeader(500)

	errTxt := fmt.Sprintf("Tests faiure: %v", err)
	log.Println(errTxt)

	_, err = w.Write([]byte(errTxt))
	if err != nil {
		log.Println(err)
	}
}

func handleSuccess(w http.ResponseWriter) {
	w.WriteHeader(200)

	successTxt :="Tests success"
	log.Println(successTxt)

	_, err := w.Write([]byte(successTxt))
	if err != nil {
		log.Println(err)
	}
}

func parseScript() (tests []test, err error) {
	jsonFile, err := os.Open(scriptPath)
	if err != nil {
		return tests, err
	}
	defer jsonFile.Close()

	bytesValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return tests, err
	}

	err = json.Unmarshal(bytesValue, &tests)
	if err != nil {
		return tests, fmt.Errorf("failed to parse '%s' as json tests: %v", string(bytesValue), err)
	}

	return
}

func parseExpectedCodesRange(input string) (maxRespCode, minResponseCode int, err error) {
	if input == "" {
		return 399, 199, nil
	}

	parts := strings.Split(input, "-")
	minResponseCode, err = strconv.Atoi(parts[0])
	if err != nil {
		return
	}

	if len(parts) == 1 {
		maxRespCode = minResponseCode
		return
	}

	maxRespCode, err = strconv.Atoi(parts[1])
	if err != nil {
		return
	}

	return
}
