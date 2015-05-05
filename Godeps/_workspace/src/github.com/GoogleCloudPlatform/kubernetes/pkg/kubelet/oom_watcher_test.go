/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package kubelet

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubelet/cadvisor"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type fakeEvent struct {
	object    runtime.Object
	timestamp util.Time
	reason    string
	message   string
}

type fakeRecorder struct {
	events []fakeEvent
}

func (f fakeRecorder) Event(object runtime.Object, reason, message string) {
	f.events = append(f.events, fakeEvent{object, util.Now(), reason, message})
}

func (f fakeRecorder) Eventf(object runtime.Object, reason, messageFmt string, args ...interface{}) {
	f.events = append(f.events, fakeEvent{object, util.Now(), reason, fmt.Sprintf(messageFmt, args...)})
}

func (f fakeRecorder) PastEventf(object runtime.Object, timestamp util.Time, reason, messageFmt string, args ...interface{}) {
	f.events = append(f.events, fakeEvent{object, timestamp, reason, fmt.Sprintf(messageFmt, args...)})
}

func TestBasic(t *testing.T) {
	fakeRecorder := fakeRecorder{}
	mockCadvisor := &cadvisor.Fake{}
	node := &api.ObjectReference{}
	oomWatcher := NewOOMWatcher(mockCadvisor, fakeRecorder)
	go func() {
		oomWatcher.RecordSysOOMs(node)
	}()
	// TODO: Improve this test once cadvisor exports events.EventChannel as an interface
	// and thereby allow using a mock version of cadvisor.
}
