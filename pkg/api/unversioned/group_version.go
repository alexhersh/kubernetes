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

package unversioned

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GroupResource specifies a Group and a Resource, but does not force a version.  This is useful for identifying
// concepts during lookup stages without having partially valid types
type GroupResource struct {
	Group    string
	Resource string
}

func (gr *GroupResource) String() string {
	return strings.Join([]string{gr.Group, ", Resource=", gr.Resource}, "")
}

// GroupVersionResource unambiguously identifies a resource.  It doesn't anonymously include GroupVersion
// to avoid automatic coersion.  It doesn't use a GroupVersion to avoid custom marshalling
type GroupVersionResource struct {
	Group    string
	Version  string
	Resource string
}

func (gvr GroupVersionResource) GroupResource() GroupResource {
	return GroupResource{Group: gvr.Group, Resource: gvr.Resource}
}

func (gvr GroupVersionResource) GroupVersion() GroupVersion {
	return GroupVersion{Group: gvr.Group, Version: gvr.Version}
}

func (gvr *GroupVersionResource) String() string {
	return strings.Join([]string{gvr.Group, "/", gvr.Version, ", Resource=", gvr.Resource}, "")
}

// GroupKind specifies a Group and a Kind, but does not force a version.  This is useful for identifying
// concepts during lookup stages without having partially valid types
//
// +protobuf.options.(gogoproto.goproto_stringer)=false
type GroupKind struct {
	Group string
	Kind  string
}

func (gk GroupKind) WithVersion(version string) GroupVersionKind {
	return GroupVersionKind{Group: gk.Group, Version: version, Kind: gk.Kind}
}

func (gk *GroupKind) String() string {
	return gk.Group + ", Kind=" + gk.Kind
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coersion.  It doesn't use a GroupVersion to avoid custom marshalling
//
// +protobuf.options.(gogoproto.goproto_stringer)=false
type GroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

// IsEmpty returns true if group, version, and kind are empty
func (gvk GroupVersionKind) IsEmpty() bool {
	return len(gvk.Group) == 0 && len(gvk.Version) == 0 && len(gvk.Kind) == 0
}

func (gvk GroupVersionKind) GroupKind() GroupKind {
	return GroupKind{Group: gvk.Group, Kind: gvk.Kind}
}

func (gvk GroupVersionKind) GroupVersion() GroupVersion {
	return GroupVersion{Group: gvk.Group, Version: gvk.Version}
}

func (gvk GroupVersionKind) String() string {
	return gvk.Group + "/" + gvk.Version + ", Kind=" + gvk.Kind
}

// GroupVersion contains the "group" and the "version", which uniquely identifies the API.
//
// +protobuf.options.(gogoproto.goproto_stringer)=false
type GroupVersion struct {
	Group   string
	Version string
}

// IsEmpty returns true if group and version are empty
func (gv GroupVersion) IsEmpty() bool {
	return len(gv.Group) == 0 && len(gv.Version) == 0
}

// String puts "group" and "version" into a single "group/version" string. For the legacy v1
// it returns "v1".
func (gv GroupVersion) String() string {
	// special case the internal apiVersion for the legacy kube types
	if gv.IsEmpty() {
		return ""
	}

	// special case of "v1" for backward compatibility
	if gv.Group == "" && gv.Version == "v1" {
		return gv.Version
	} else {
		return gv.Group + "/" + gv.Version
	}
}

// ParseGroupVersion turns "group/version" string into a GroupVersion struct. It reports error
// if it cannot parse the string.
func ParseGroupVersion(gv string) (GroupVersion, error) {
	// this can be the internal version for the legacy kube types
	// TODO once we've cleared the last uses as strings, this special case should be removed.
	if (len(gv) == 0) || (gv == "/") {
		return GroupVersion{}, nil
	}

	s := strings.Split(gv, "/")
	// "v1" is the only special case. Otherwise GroupVersion is expected to contain
	// one "/" dividing the string into two parts.
	switch {
	case len(s) == 1 && gv == "v1":
		return GroupVersion{"", "v1"}, nil
	case len(s) == 2:
		return GroupVersion{s[0], s[1]}, nil
	default:
		return GroupVersion{}, fmt.Errorf("Unexpected GroupVersion string: %v", gv)
	}
}

func ParseGroupVersionOrDie(gv string) GroupVersion {
	ret, err := ParseGroupVersion(gv)
	if err != nil {
		panic(err)
	}

	return ret
}

// WithKind creates a GroupVersionKind based on the method receiver's GroupVersion and the passed Kind.
func (gv GroupVersion) WithKind(kind string) GroupVersionKind {
	return GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}
}

// WithResource creates a GroupVersionResource based on the method receiver's GroupVersion and the passed Resource.
func (gv GroupVersion) WithResource(resource string) GroupVersionResource {
	return GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource}
}

// MarshalJSON implements the json.Marshaller interface.
func (gv GroupVersion) MarshalJSON() ([]byte, error) {
	s := gv.String()
	if strings.Count(s, "/") > 1 {
		return []byte{}, fmt.Errorf("illegal GroupVersion %v: contains more than one /", s)
	}
	return json.Marshal(s)
}

func (gv *GroupVersion) unmarshal(value []byte) error {
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return err
	}
	parsed, err := ParseGroupVersion(s)
	if err != nil {
		return err
	}
	*gv = parsed
	return nil
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (gv *GroupVersion) UnmarshalJSON(value []byte) error {
	return gv.unmarshal(value)
}

// UnmarshalTEXT implements the Ugorji's encoding.TextUnmarshaler interface.
func (gv *GroupVersion) UnmarshalText(value []byte) error {
	return gv.unmarshal(value)
}

// ToAPIVersionAndKind is a convenience method for satisfying runtime.Object on types that
// do not use TypeMeta.
func (gvk *GroupVersionKind) ToAPIVersionAndKind() (string, string) {
	if gvk == nil {
		return "", ""
	}
	return gvk.GroupVersion().String(), gvk.Kind
}

// FromAPIVersionAndKind returns a GVK representing the provided fields for types that
// do not use TypeMeta. This method exists to support test types and legacy serializations
// that have a distinct group and kind.
// TODO: further reduce usage of this method.
func FromAPIVersionAndKind(apiVersion, kind string) *GroupVersionKind {
	if gv, err := ParseGroupVersion(apiVersion); err == nil {
		return &GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}
	}
	return &GroupVersionKind{Kind: kind}
}

// All objects that are serialized from a Scheme encode their type information. This interface is used
// by serialization to set type information from the Scheme onto the serialized version of an object.
// For objects that cannot be serialized or have unique requirements, this interface may be a no-op.
// TODO: this belongs in pkg/runtime, move unversioned.GVK into runtime.
type ObjectKind interface {
	// SetGroupVersionKind sets or clears the intended serialized kind of an object. Passing kind nil
	// should clear the current setting.
	SetGroupVersionKind(kind *GroupVersionKind)
	// GroupVersionKind returns the stored group, version, and kind of an object, or nil if the object does
	// not expose or provide these fields.
	GroupVersionKind() *GroupVersionKind
}

// EmptyObjectKind implements the ObjectKind interface as a noop
// TODO: this belongs in pkg/runtime, move unversioned.GVK into runtime.
var EmptyObjectKind = emptyObjectKind{}

type emptyObjectKind struct{}

// SetGroupVersionKind implements the ObjectKind interface
func (emptyObjectKind) SetGroupVersionKind(gvk *GroupVersionKind) {}

// GroupVersionKind implements the ObjectKind interface
func (emptyObjectKind) GroupVersionKind() *GroupVersionKind { return nil }
