/*
Copyright 2025 Erfan Mahvash.

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

package v1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EifaReplicaSpec defines the desired state of EifaReplica
type ScaleTargetRef struct {
	// +kubebuilder:validation:Enum={"Deployment","deployment","deploy","Deploy"}
	Kind string `json:"kind"`
	Name string `json:"name"`
}
type EifaReplicaSpec struct {
	ScaleTargetRef ScaleTargetRef `json:"scaleTargetRef"`

	// +kubebuilder:validation:Minimum=0
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// +kubebuilder:validation:Minimum=0
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// +kubebuilder:validation:Pattern=`^(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\d+(ns|us|µs|ms|s|m|h))+)|((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5,7})$`
	Schedule    string                  `json:"schedule"`
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate" protobuf:"bytes,1,opt,name=jobTemplate"`
}

const (
	JOB_SUCCESS = "Job-Success"
	JOB_FAILED  = "Job-Failed"
	JOB_RUNNING = "Job-Running"
	FAILED      = "Failed"
	SUCCESS     = "Success"
)

// EifaReplicaStatus defines the observed state of EifaReplica
type EifaReplicaStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	NextTransitionTime string             `json:"nextTransitionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=er

// EifaReplica is the Schema for the eifareplicas API
type EifaReplica struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EifaReplicaSpec   `json:"spec,omitempty"`
	Status EifaReplicaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EifaReplicaList contains a list of EifaReplica
type EifaReplicaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EifaReplica `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EifaReplica{}, &EifaReplicaList{})
}
