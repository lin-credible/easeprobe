/*
 * Copyright (c) 2022, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/megaease/easeprobe/global"
	"github.com/megaease/easeprobe/probe/base"
	log "github.com/sirupsen/logrus"
)

// HTTP implements a config for HTTP.
type HTTP struct {
	base.DefaultOptions `yaml:",inline"`
	URL                 string            `yaml:"url"`
	ContentEncoding     string            `yaml:"content_encoding,omitempty"`
	Method              string            `yaml:"method,omitempty"`
	Headers             map[string]string `yaml:"headers,omitempty"`
	Body                string            `yaml:"body,omitempty"`

	//Option - HTTP Basic Auth Credentials
	User string `yaml:"username,omitempty"`
	Pass string `yaml:"password,omitempty"`

	//Option - TLS Config
	global.TLS `yaml:",inline"`

	client *http.Client `yaml:"-"`
}

func checkHTTPMethod(m string) bool {

	methods := [...]string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE"}
	for _, method := range methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// Config HTTP Config Object
func (h *HTTP) Config(gConf global.ProbeSettings) error {

	kind := "http"
	tag := ""
	name := h.ProbeName
	h.DefaultOptions.Config(gConf, kind, tag, name, h.URL, h.DoProbe)

	if _, err := url.ParseRequestURI(h.URL); err != nil {
		log.Errorf("URL is not valid - %+v url=%+v", err)
		return err
	}

	tls, err := h.TLS.Config()
	if err != nil {
		log.Errorf("TLS configuration error - %s", err)
		return err
	}

	h.client = &http.Client{
		Timeout: h.Timeout(),
		Transport: &http.Transport{
			TLSClientConfig: tls,
		},
	}
	if !checkHTTPMethod(h.Method) {
		h.Method = "GET"
	}

	log.Debugf("[%s] configuration: %+v, %+v", h.ProbeKind, h, h.Result())
	return nil
}

// DoProbe return the checking result
func (h *HTTP) DoProbe() (bool, string) {

	req, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer([]byte(h.Body)))
	if err != nil {
		return false, fmt.Sprintf("HTTP request error - %v", err)
	}
	if len(h.User) > 0 && len(h.Pass) > 0 {
		req.SetBasicAuth(h.User, h.Pass)
	}
	if len(h.ContentEncoding) > 0 {
		req.Header.Set("Content-Type", h.ContentEncoding)
	}
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}

	// client close the connection
	req.Close = true

	req.Header.Set("User-Agent", global.OrgProgVer)
	resp, err := h.client.Do(req)

	status := true
	message := ""
	if err != nil {
		message = fmt.Sprintf("Error: %v", err)
		log.Errorf("error making get request: %v", err)
		status = false
	} else {
		// Read the response body
		defer resp.Body.Close()
		response, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Debugf("%s", string(response))
		}
		message = fmt.Sprintf("HTTP Status Code is %d", resp.StatusCode)
		if resp.StatusCode >= 500 {
			message = fmt.Sprintf("HTTP Status Code is %d", resp.StatusCode)
			status = false
		}
	}

	return status, message
}
