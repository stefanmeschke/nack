// Copyright 2020 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1beta2 "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	jetstreamv1beta2 "github.com/nats-io/nack/pkg/jetstream/generated/applyconfiguration/jetstream/v1beta2"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=jetstream.nats.io, Version=v1beta2
	case v1beta2.SchemeGroupVersion.WithKind("Account"):
		return &jetstreamv1beta2.AccountApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("AccountSpec"):
		return &jetstreamv1beta2.AccountSpecApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("Condition"):
		return &jetstreamv1beta2.ConditionApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("Consumer"):
		return &jetstreamv1beta2.ConsumerApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("ConsumerSpec"):
		return &jetstreamv1beta2.ConsumerSpecApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("CredsSecret"):
		return &jetstreamv1beta2.CredsSecretApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("RePublish"):
		return &jetstreamv1beta2.RePublishApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("SecretRef"):
		return &jetstreamv1beta2.SecretRefApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("Status"):
		return &jetstreamv1beta2.StatusApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("Stream"):
		return &jetstreamv1beta2.StreamApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("StreamPlacement"):
		return &jetstreamv1beta2.StreamPlacementApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("StreamSource"):
		return &jetstreamv1beta2.StreamSourceApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("StreamSpec"):
		return &jetstreamv1beta2.StreamSpecApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("SubjectTransform"):
		return &jetstreamv1beta2.SubjectTransformApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("TLS"):
		return &jetstreamv1beta2.TLSApplyConfiguration{}
	case v1beta2.SchemeGroupVersion.WithKind("TLSSecret"):
		return &jetstreamv1beta2.TLSSecretApplyConfiguration{}

	}
	return nil
}
