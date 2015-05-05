/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/latest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/testapi"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type fakeRL bool

func (fakeRL) Stop()             {}
func (f fakeRL) CanAccept() bool { return bool(f) }
func (f fakeRL) Accept()         {}

func expectHTTP(url string, code int, t *testing.T) {
	r, err := http.Get(url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r.StatusCode != code {
		t.Errorf("unexpected response: %v", r.StatusCode)
	}
}

func getPath(resource, namespace, name string) string {
	return testapi.ResourcePath(resource, namespace, name)
}

func pathWithNamespaceQuery(resource, namespace, name string) string {
	return testapi.ResourcePathWithNamespaceQuery(resource, namespace, name)
}

func pathWithPrefix(prefix, resource, namespace, name string) string {
	return testapi.ResourcePathWithPrefix(prefix, resource, namespace, name)
}

func pathWithPrefixAndNamespaceQuery(prefix, resource, namespace, name string) string {
	return testapi.ResourcePathWithPrefixAndNamespaceQuery(prefix, resource, namespace, name)
}

func TestMaxInFlight(t *testing.T) {
	const Iterations = 3
	block := sync.WaitGroup{}
	block.Add(1)
	sem := make(chan bool, Iterations)

	re := regexp.MustCompile("[.*\\/watch][^\\/proxy.*]")

	server := httptest.NewServer(MaxInFlightLimit(sem, re, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "dontwait") {
			return
		}
		block.Wait()
	})))
	defer server.Close()

	// These should hang, but not affect accounting.
	for i := 0; i < Iterations; i++ {
		// These should hang waiting on block...
		go func() {
			expectHTTP(server.URL+"/foo/bar/watch", http.StatusOK, t)
		}()
	}
	for i := 0; i < Iterations; i++ {
		// These should hang waiting on block...
		go func() {
			expectHTTP(server.URL+"/proxy/foo/bar", http.StatusOK, t)
		}()
	}
	expectHTTP(server.URL+"/dontwait", http.StatusOK, t)

	for i := 0; i < Iterations; i++ {
		// These should hang waiting on block...
		go func() {
			expectHTTP(server.URL, http.StatusOK, t)
		}()
	}
	// There's really no more elegant way to do this.  I could use a WaitGroup, but even then
	// it'd still be racy.
	time.Sleep(1 * time.Second)
	expectHTTP(server.URL+"/dontwait/watch", http.StatusOK, t)

	// Do this multiple times to show that it rate limit rejected requests don't block.
	for i := 0; i < 2; i++ {
		expectHTTP(server.URL, errors.StatusTooManyRequests, t)
	}
	block.Done()

	// Show that we recover from being blocked up.
	expectHTTP(server.URL, http.StatusOK, t)
}

func TestRateLimit(t *testing.T) {
	for _, allow := range []bool{true, false} {
		rl := fakeRL(allow)
		server := httptest.NewServer(RateLimit(rl, http.HandlerFunc(
			func(w http.ResponseWriter, req *http.Request) {
				if !allow {
					t.Errorf("Unexpected call")
				}
			},
		)))
		defer server.Close()
		http.Get(server.URL)
	}
}

func TestReadOnly(t *testing.T) {
	server := httptest.NewServer(ReadOnly(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "GET" {
				t.Errorf("Unexpected call: %v", req.Method)
			}
		},
	)))
	defer server.Close()
	for _, verb := range []string{"GET", "POST", "PUT", "DELETE", "CREATE"} {
		req, err := http.NewRequest(verb, server.URL, nil)
		if err != nil {
			t.Fatalf("Couldn't make request: %v", err)
		}
		http.DefaultClient.Do(req)
	}
}

func TestGetAPIRequestInfo(t *testing.T) {
	successCases := []struct {
		method              string
		url                 string
		expectedVerb        string
		expectedAPIVersion  string
		expectedNamespace   string
		expectedResource    string
		expectedSubresource string
		expectedKind        string
		expectedName        string
		expectedParts       []string
	}{

		// resource paths
		{"GET", "/namespaces", "list", "", "", "namespaces", "", "Namespace", "", []string{"namespaces"}},
		{"GET", "/namespaces/other", "get", "", "other", "namespaces", "", "Namespace", "other", []string{"namespaces", "other"}},

		{"GET", "/namespaces/other/pods", "list", "", "other", "pods", "", "Pod", "", []string{"pods"}},
		{"GET", "/namespaces/other/pods/foo", "get", "", "other", "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/pods", "list", "", api.NamespaceAll, "pods", "", "Pod", "", []string{"pods"}},
		{"POST", "/pods", "create", "", api.NamespaceDefault, "pods", "", "Pod", "", []string{"pods"}},
		{"GET", "/pods/foo", "get", "", api.NamespaceDefault, "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/pods/foo?namespace=other", "get", "", "other", "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/pods?namespace=other", "list", "", "other", "pods", "", "Pod", "", []string{"pods"}},

		// special verbs
		{"GET", "/proxy/namespaces/other/pods/foo", "proxy", "", "other", "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/proxy/pods/foo", "proxy", "", api.NamespaceDefault, "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/redirect/namespaces/other/pods/foo", "redirect", "", "other", "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/redirect/pods/foo", "redirect", "", api.NamespaceDefault, "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", "/watch/pods", "watch", "", api.NamespaceAll, "pods", "", "Pod", "", []string{"pods"}},
		{"GET", "/watch/namespaces/other/pods", "watch", "", "other", "pods", "", "Pod", "", []string{"pods"}},

		// fully-qualified paths
		{"GET", pathWithNamespaceQuery("pods", "other", ""), "list", testapi.Version(), "other", "pods", "", "Pod", "", []string{"pods"}},
		{"GET", pathWithNamespaceQuery("pods", "other", "foo"), "get", testapi.Version(), "other", "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", getPath("pods", "", ""), "list", testapi.Version(), api.NamespaceAll, "pods", "", "Pod", "", []string{"pods"}},
		{"POST", getPath("pods", "", ""), "create", testapi.Version(), api.NamespaceDefault, "pods", "", "Pod", "", []string{"pods"}},
		{"GET", getPath("pods", "", "foo"), "get", testapi.Version(), api.NamespaceDefault, "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", pathWithPrefix("proxy", "pods", "", "foo"), "proxy", testapi.Version(), api.NamespaceDefault, "pods", "", "Pod", "foo", []string{"pods", "foo"}},
		{"GET", pathWithPrefix("watch", "pods", "", ""), "watch", testapi.Version(), api.NamespaceAll, "pods", "", "Pod", "", []string{"pods"}},
		{"GET", pathWithPrefixAndNamespaceQuery("redirect", "pods", "", ""), "redirect", testapi.Version(), api.NamespaceAll, "pods", "", "Pod", "", []string{"pods"}},
		{"GET", pathWithPrefixAndNamespaceQuery("watch", "pods", "other", ""), "watch", testapi.Version(), "other", "pods", "", "Pod", "", []string{"pods"}},

		// subresource identification
		{"GET", "/namespaces/other/pods/foo/status", "get", "", "other", "pods", "status", "Pod", "foo", []string{"pods", "foo", "status"}},
		{"PUT", "/namespaces/other/finalize", "update", "", "other", "finalize", "", "", "", []string{"finalize"}},
	}

	apiRequestInfoResolver := &APIRequestInfoResolver{util.NewStringSet("api"), latest.RESTMapper}

	for _, successCase := range successCases {
		req, _ := http.NewRequest(successCase.method, successCase.url, nil)

		apiRequestInfo, err := apiRequestInfoResolver.GetAPIRequestInfo(req)
		if err != nil {
			t.Errorf("Unexpected error for url: %s %v", successCase.url, err)
		}
		if successCase.expectedVerb != apiRequestInfo.Verb {
			t.Errorf("Unexpected verb for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedVerb, apiRequestInfo.Verb)
		}
		if successCase.expectedAPIVersion != apiRequestInfo.APIVersion {
			t.Errorf("Unexpected apiVersion for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedAPIVersion, apiRequestInfo.APIVersion)
		}
		if successCase.expectedNamespace != apiRequestInfo.Namespace {
			t.Errorf("Unexpected namespace for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedNamespace, apiRequestInfo.Namespace)
		}
		if successCase.expectedKind != apiRequestInfo.Kind {
			t.Errorf("Unexpected kind for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedKind, apiRequestInfo.Kind)
		}
		if successCase.expectedResource != apiRequestInfo.Resource {
			t.Errorf("Unexpected resource for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedResource, apiRequestInfo.Resource)
		}
		if successCase.expectedSubresource != apiRequestInfo.Subresource {
			t.Errorf("Unexpected resource for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedSubresource, apiRequestInfo.Subresource)
		}
		if successCase.expectedName != apiRequestInfo.Name {
			t.Errorf("Unexpected name for url: %s, expected: %s, actual: %s", successCase.url, successCase.expectedName, apiRequestInfo.Name)
		}
		if !reflect.DeepEqual(successCase.expectedParts, apiRequestInfo.Parts) {
			t.Errorf("Unexpected parts for url: %s, expected: %v, actual: %v", successCase.url, successCase.expectedParts, apiRequestInfo.Parts)
		}
	}

	errorCases := map[string]string{
		"no resource path":            "/",
		"just apiversion":             "/api/version/",
		"apiversion with no resource": "/api/version/",
	}
	for k, v := range errorCases {
		req, err := http.NewRequest("GET", v, nil)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		_, err = apiRequestInfoResolver.GetAPIRequestInfo(req)
		if err == nil {
			t.Errorf("Expected error for key: %s", k)
		}
	}
}
