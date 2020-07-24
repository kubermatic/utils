/*
Copyright 2020 The Kubermatic Authors.

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

package util

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLogger(t *testing.T) {
	var buf bytes.Buffer
	log := BuildLogger(7, false, &buf)
	log.V(8).Info("hello")
	assert.Empty(t, buf.Bytes())
	log.V(6).Info("hello")
	assert.NotEmpty(t, buf.Bytes())
	v := map[string]interface{}{}
	err := json.Unmarshal(buf.Bytes(), &v)
	assert.Nil(t, err)
	buf.Reset()

	log = BuildLogger(7, true, &buf)
	log.V(8).Info("hello")
	assert.Empty(t, buf.Bytes())
	log.V(6).Info("hello")
	assert.NotEmpty(t, buf.Bytes())
	err = json.Unmarshal(buf.Bytes(), &v)
	assert.NotNil(t, err)
}
