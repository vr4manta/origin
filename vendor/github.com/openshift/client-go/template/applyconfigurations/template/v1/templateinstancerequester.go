// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	templatev1 "github.com/openshift/api/template/v1"
)

// TemplateInstanceRequesterApplyConfiguration represents a declarative configuration of the TemplateInstanceRequester type for use
// with apply.
type TemplateInstanceRequesterApplyConfiguration struct {
	Username *string                          `json:"username,omitempty"`
	UID      *string                          `json:"uid,omitempty"`
	Groups   []string                         `json:"groups,omitempty"`
	Extra    map[string]templatev1.ExtraValue `json:"extra,omitempty"`
}

// TemplateInstanceRequesterApplyConfiguration constructs a declarative configuration of the TemplateInstanceRequester type for use with
// apply.
func TemplateInstanceRequester() *TemplateInstanceRequesterApplyConfiguration {
	return &TemplateInstanceRequesterApplyConfiguration{}
}

// WithUsername sets the Username field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Username field is set to the value of the last call.
func (b *TemplateInstanceRequesterApplyConfiguration) WithUsername(value string) *TemplateInstanceRequesterApplyConfiguration {
	b.Username = &value
	return b
}

// WithUID sets the UID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UID field is set to the value of the last call.
func (b *TemplateInstanceRequesterApplyConfiguration) WithUID(value string) *TemplateInstanceRequesterApplyConfiguration {
	b.UID = &value
	return b
}

// WithGroups adds the given value to the Groups field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Groups field.
func (b *TemplateInstanceRequesterApplyConfiguration) WithGroups(values ...string) *TemplateInstanceRequesterApplyConfiguration {
	for i := range values {
		b.Groups = append(b.Groups, values[i])
	}
	return b
}

// WithExtra puts the entries into the Extra field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Extra field,
// overwriting an existing map entries in Extra field with the same key.
func (b *TemplateInstanceRequesterApplyConfiguration) WithExtra(entries map[string]templatev1.ExtraValue) *TemplateInstanceRequesterApplyConfiguration {
	if b.Extra == nil && len(entries) > 0 {
		b.Extra = make(map[string]templatev1.ExtraValue, len(entries))
	}
	for k, v := range entries {
		b.Extra[k] = v
	}
	return b
}
