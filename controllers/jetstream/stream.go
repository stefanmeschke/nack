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

package jetstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta1"

	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	streamFinalizerKey = "streamfinalizer.jetstream.nats.io"
)

func (c *Controller) runStreamQueue() {
	for {
		processQueueNext(c.strQueue, &realJsmClient{}, c.processStream)
	}
}

func (c *Controller) processStream(ns, name string, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process stream: %w", err)
		}
	}()

	str, err := c.strLister.Streams(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	spec := str.Spec
	ifc := c.ji.Streams(str.Namespace)

	defer func() {
		if err == nil {
			return
		}

		if _, serr := setStreamErrored(c.ctx, str, ifc, err); serr != nil {
			err = fmt.Errorf("%s: %w", err, serr)
		}
	}()

	creds, err := getCreds(c.ctx, spec.CredentialsSecret, c.ki.Secrets(ns))
	if err != nil {
		return err
	}

	c.normalEvent(str, "Connecting", "Connecting to NATS Server")
	err = jsmc.Connect(
		strings.Join(spec.Servers, ","),
		getNATSOptions(c.natsName, creds)...,
	)
	if err != nil {
		return err
	}
	defer jsmc.Close()
	c.normalEvent(str, "Connected", "Connected to NATS Server")

	deleteOK := str.GetDeletionTimestamp() != nil
	newGeneration := str.Generation != str.Status.ObservedGeneration
	strOK, err := streamExists(c.ctx, jsmc, spec.Name)
	if err != nil {
		return err
	}
	updateOK := (strOK && !deleteOK && newGeneration)
	createOK := (!strOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		c.normalEvent(str, "Creating", fmt.Sprintf("Creating stream %q", spec.Name))
		if err := createStream(c.ctx, jsmc, spec); err != nil {
			return err
		}

		res, err := setStreamFinalizer(c.ctx, str, ifc)
		if err != nil {
			return err
		}
		str = res

		if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
			return err
		}
		c.normalEvent(str, "Created", fmt.Sprintf("Created stream %q", spec.Name))
	case updateOK:
		c.normalEvent(str, "Updating", fmt.Sprintf("Updating stream %q", spec.Name))
		if err := updateStream(c.ctx, jsmc, spec); err != nil {
			return err
		}

		res, err := setStreamFinalizer(c.ctx, str, ifc)
		if err != nil {
			return err
		}
		str = res

		if _, err := setStreamOK(c.ctx, str, ifc); err != nil {
			return err
		}
		c.normalEvent(str, "Updated", fmt.Sprintf("Updated stream %q", spec.Name))
		return nil
	case deleteOK:
		c.normalEvent(str, "Deleting", fmt.Sprintf("Deleting stream %q", spec.Name))
		if err := deleteStream(c.ctx, jsmc, spec.Name); err != nil {
			return err
		}

		if _, err := clearStreamFinalizer(c.ctx, str, ifc); err != nil {
			return err
		}
	default:
		c.warningEvent(str, "Noop", fmt.Sprintf("Nothing done for stream %q", spec.Name))
	}

	return nil
}

func streamExists(ctx context.Context, c jsmClient, name string) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	var apierr jsmapi.ApiError
	_, err = c.LoadStream(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func createStream(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create stream %q: %w", spec.Name, err)
		}
	}()

	maxAge, err := getMaxAge(spec.MaxAge)
	if err != nil {
		return err
	}

	retention := getRetention(spec.Retention)
	storage := getStorage(spec.Storage)
	discard := getDiscard(spec.Discard)

	duplicates, err := getDuplicates(spec.DuplicateWindow)
	if err != nil {
		return err
	}

	_, err = c.NewStream(ctx, jsmapi.StreamConfig{
		Name:         spec.Name,
		Retention:    retention,
		Subjects:     spec.Subjects,
		MaxConsumers: spec.MaxConsumers,
		MaxMsgs:      int64(spec.MaxMsgs),
		MaxBytes:     int64(spec.MaxBytes),
		MaxAge:       maxAge,
		MaxMsgSize:   int32(spec.MaxMsgSize),
		Storage:      storage,
		Discard:      discard,
		Replicas:     spec.Replicas,
		NoAck:        spec.NoAck,
		Duplicates:   duplicates,
	})
	return err
}

func updateStream(ctx context.Context, c jsmClient, spec apis.StreamSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to update stream %q: %w", spec.Name, err)
		}
	}()

	js, err := c.LoadStream(ctx, spec.Name)
	if err != nil {
		return err
	}

	maxAge, err := getMaxAge(spec.MaxAge)
	if err != nil {
		return err
	}

	retention := getRetention(spec.Retention)
	storage := getStorage(spec.Storage)
	discard := getDiscard(spec.Discard)

	duplicates, err := getDuplicates(spec.DuplicateWindow)
	if err != nil {
		return err
	}

	return js.UpdateConfiguration(jsmapi.StreamConfig{
		Name:         spec.Name,
		Retention:    retention,
		Subjects:     spec.Subjects,
		MaxConsumers: spec.MaxConsumers,
		MaxMsgs:      int64(spec.MaxMsgs),
		MaxBytes:     int64(spec.MaxBytes),
		MaxAge:       maxAge,
		MaxMsgSize:   int32(spec.MaxMsgSize),
		Storage:      storage,
		Discard:      discard,
		Replicas:     spec.Replicas,
		NoAck:        spec.NoAck,
		Duplicates:   duplicates,
	})
}

func deleteStream(ctx context.Context, c jsmClient, name string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream %q: %w", name, err)
		}
	}()

	var apierr jsmapi.ApiError
	str, err := c.LoadStream(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return str.Delete()
}

func setStreamErrored(ctx context.Context, s *apis.Stream, sif typed.StreamInterface, err error) (*apis.Stream, error) {
	if err == nil {
		return s, nil
	}

	sc := s.DeepCopy()
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            err.Error(),
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := sif.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream errored status: %w", err)
	}

	return res, nil
}

func setStreamOK(ctx context.Context, s *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Created",
		Message:            "Stream successfully created",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream %q status: %w", s.Spec.Name, err)
	}

	return res, nil
}

func setStreamFinalizer(ctx context.Context, o *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	o.SetFinalizers(addFinalizer(o.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, o, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set %q stream finalizers: %w", o.GetName(), err)
	}

	return res, nil
}

func clearStreamFinalizer(ctx context.Context, o *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	o.SetFinalizers(removeFinalizer(o.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, o, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear %q stream finalizers: %w", o.GetName(), err)
	}

	return res, nil
}

func getMaxAge(v string) (time.Duration, error) {
	if v == "" {
		return time.Duration(0), nil
	}

	return time.ParseDuration(v)
}

func getRetention(v string) jsmapi.RetentionPolicy {
	retention := jsmapi.LimitsPolicy
	switch v {
	case "interest":
		retention = jsmapi.InterestPolicy
	case "workqueue":
		retention = jsmapi.WorkQueuePolicy
	}
	return retention
}

func getStorage(v string) jsmapi.StorageType {
	storage := jsmapi.MemoryStorage
	switch v {
	case "file":
		storage = jsmapi.FileStorage
	}
	return storage
}

func getDiscard(v string) jsmapi.DiscardPolicy {
	discard := jsmapi.DiscardOld
	switch v {
	case "new":
		discard = jsmapi.DiscardNew
	}
	return discard
}

func getDuplicates(v string) (time.Duration, error) {
	if v == "" {
		return time.Duration(0), nil
	}

	return time.ParseDuration(v)
}
