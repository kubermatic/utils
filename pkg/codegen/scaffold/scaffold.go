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

package scaffold

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	scaffoldTemplateRaw = `
package {{.Resource.Version}}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {

	// INSERT ADDITIONAL SPEC FIELDS -- desired state of cluster

{{- if .Spec.Metadata }}
	// Metadata	contains additional human readable {{.Resource.Kind}} details.
	Metadata *{{.Resource.Kind}}Metadata ` + "`" + `json:"metadata,omitempty"` + "`" + `
{{ end }}
}

{{- if .Spec.Metadata }}
// {{.Resource.Kind}}Metadata contains the metadata of the {{.Resource.Kind}}.
type {{.Resource.Kind}}Metadata struct {
	// DisplayName is the human-readable name of this {{.Resource.Kind}}.
	// +kubebuilder:validation:MinLength=1
	DisplayName string ` + "`" + `json:"displayName"` + "`" + `
	// Description is the long and detailed description of the {{.Resource.Kind}}.
	// +kubebuilder:validation:MinLength=1
	Description string ` + "`" + `json:"description"` + "`" + `
}
{{ end }}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}.
// It should always be reconstructable from the state of the cluster and/or outside world.
type {{.Resource.Kind}}Status struct {

	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster

	// ObservedGeneration is the most recent generation observed for this {{.Resource.Kind}} by the controller.
	ObservedGeneration int64 ` + "`" + `json:"observedGeneration,omitempty"` + "`" + `
	// Conditions represents the latest available observations of a {{.Resource.Kind}}'s current state.
	Conditions []{{.Resource.Kind}}Condition ` + "`" + `json:"conditions,omitempty"` + "`" + `
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase {{.Resource.Kind}}PhaseType ` + "`" + `json:"phase,omitempty"` + "`" + `
}

// {{.Resource.Kind}}PhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type {{.Resource.Kind}}PhaseType string

// Values of {{.Resource.Kind}}PhaseType.
const (
	{{.Resource.Kind}}PhaseReady       {{.Resource.Kind}}PhaseType = "Ready"
	{{.Resource.Kind}}PhaseNotReady    {{.Resource.Kind}}PhaseType = "NotReady"
	{{.Resource.Kind}}PhaseUnknown     {{.Resource.Kind}}PhaseType = "Unknown"
	{{.Resource.Kind}}PhaseTerminating {{.Resource.Kind}}PhaseType = "Terminating"
)

const (
	{{.Resource.Kind}}TerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *{{.Resource.Kind}}Status) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != {{.Resource.Kind}}Ready {
			continue
		}

		switch condition.Status {
		case {{.Resource.Kind}}ConditionTrue:
			s.Phase = {{.Resource.Kind}}PhaseReady
		case {{.Resource.Kind}}ConditionFalse:
			if condition.Reason == {{.Resource.Kind}}TerminatingReason {
				s.Phase = {{.Resource.Kind}}PhaseTerminating
			} else {
				s.Phase = {{.Resource.Kind}}PhaseNotReady
			}
		case {{.Resource.Kind}}ConditionUnknown:
			s.Phase = {{.Resource.Kind}}PhaseUnknown
		}
		return
	}

	s.Phase = {{.Resource.Kind}}PhaseUnknown
}

// {{.Resource.Kind}}ConditionType represents a {{.Resource.Kind}}Condition value.
// +kubebuilder:validation:Ready
type {{.Resource.Kind}}ConditionType string

const (
	// {{.Resource.Kind}}Ready represents a {{.Resource.Kind}} condition is in ready state.
	{{.Resource.Kind}}Ready {{.Resource.Kind}}ConditionType = "Ready"
)

// {{.Resource.Kind}}ConditionStatus represents a condition's status.
// +kubebuilder:validation:True;False;Unknown
type {{.Resource.Kind}}ConditionStatus string

// These are valid condition statuses. "{{.Resource.Kind}}ConditionTrue" means a resource is in
// the condition; "{{.Resource.Kind}}ConditionFalse" means a resource is not in the condition;
// "{{.Resource.Kind}}ConditionFalse" means Kubernetes can't decide if a resource is in the
// condition or not.
const (
	// {{.Resource.Kind}}ConditionTrue represents the fact that a given condition is true
	{{.Resource.Kind}}ConditionTrue {{.Resource.Kind}}ConditionStatus = "True"

	// {{.Resource.Kind}}ConditionFalse represents the fact that a given condition is false
	{{.Resource.Kind}}ConditionFalse {{.Resource.Kind}}ConditionStatus = "False"

	// {{.Resource.Kind}}ConditionUnknown represents the fact that a given condition is unknown
	{{.Resource.Kind}}ConditionUnknown {{.Resource.Kind}}ConditionStatus = "Unknown"
)

// {{.Resource.Kind}}Condition contains details for the current condition of this {{.Resource.Kind}}.
type {{.Resource.Kind}}Condition struct {
	// Type is the type of the {{.Resource.Kind}} condition, currently ('Ready').
	Type {{.Resource.Kind}}ConditionType ` + "`" + `json:"type"` + "`" + `
	// Status is the status of the condition, one of ('True', 'False', 'Unknown').
	Status {{.Resource.Kind}}ConditionStatus ` + "`" + `json:"status"` + "`" + `
	// LastTransitionTime is the last time the condition transits from one status to another.
	LastTransitionTime metav1.Time ` + "`" + `json:"lastTransitionTime"` + "`" + `
	// Reason is the (brief) reason for the condition's last transition.
	Reason string ` + "`" + `json:"reason"` + "`" + `
	// Message is the human readable message indicating details about last transition.
	Message string ` + "`" + `json:"message"` + "`" + `
}

// GetCondition returns the Condition of the given condition type, if it exists.
func (s *{{.Resource.Kind}}Status) GetCondition(t {{.Resource.Kind}}ConditionType) (condition {{.Resource.Kind}}Condition, exists bool) {
	for _, cond := range s.Conditions {
		if cond.Type == t {
			condition = cond
			exists = true
			return
		}
	}
	return
}

// SetCondition replaces or adds the given condition.
func (s *{{.Resource.Kind}}Status) SetCondition(condition {{.Resource.Kind}}Condition) {
	defer s.updatePhase()

	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}

	for i := range s.Conditions {
		if s.Conditions[i].Type == condition.Type {

			// Only update the LastTransitionTime when the Status is changed.
			if s.Conditions[i].Status != condition.Status {
				s.Conditions[i].LastTransitionTime = condition.LastTransitionTime
			}

			s.Conditions[i].Status = condition.Status
			s.Conditions[i].Reason = condition.Reason
			s.Conditions[i].Message = condition.Message

			return
		}
	}

	s.Conditions = append(s.Conditions, condition)
}

// {{.Resource.Kind}} is the Schema for the {{.Resource.Kind}} API.
// +kubebuilder:object:root=true
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// IsReady returns if the {{.Resource.Kind}} is ready.
func (s *{{.Resource.Kind}}) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == {{.Resource.Kind}}Ready &&
			condition.Status == {{.Resource.Kind}}ConditionTrue {
			return true
		}
	}
	return false
}

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{.Resource.Kind}} ` + "`" + `json:"items"` + "`" + `
}
`
	typesTemplateHelpers = template.FuncMap{
		"SplitLines": func(raw string) []string { return strings.Split(raw, "\n") },
	}

	typesTemplate = template.Must(template.New("status-scaffolding").Funcs(typesTemplateHelpers).Parse(scaffoldTemplateRaw))
)

// ScaffoldOptions describes how to scaffold out a Kubernetes object
// with the basic metadata and comment annotations required to generate code
// for and conform to runtime.Object and metav1.Object.
type ScaffoldOptions struct {
	Resource Resource
	Spec     Spec
	Status   Status
	// The Path of the Boilerplate header file.
	Boilerplate string
	OutputPath  string
}

// Validate validates the options, returning an error if anything is invalid.
func (o *ScaffoldOptions) Validate() error {
	if err := o.Resource.Validate(); err != nil {
		return err
	}
	if o.OutputPath == "" {
		o.OutputPath = strings.ToLower(o.Resource.Kind) + "_types.go"
	}

	// Check if the file to write already exists
	if _, err := os.Stat(o.OutputPath); err == nil {
		// file is already exist
		return fmt.Errorf("%s already exists", o.OutputPath)
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}

// Scaffold prints the Kubernetes object scaffolding to the given output.
func (o *ScaffoldOptions) Scaffold() error {
	f, err := os.Create(o.OutputPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	boilerplate, err := o.loadBoilerplate()
	if err != nil {
		return err
	}
	if _, err := f.Write(boilerplate); err != nil {
		return err
	}
	return typesTemplate.Execute(f, o)
}

func (o *ScaffoldOptions) loadBoilerplate() ([]byte, error) {
	b, err := ioutil.ReadFile(o.Boilerplate)
	if err != nil {
		return nil, err
	}
	b = bytes.Replace(b, []byte("YEAR"), []byte(strconv.Itoa(time.Now().UTC().Year())), -1)
	return b, nil
}
