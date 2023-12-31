/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package certmod

import (
	"context"
	"fmt"
	"github.com/keyvalstore/store"
	"github.com/sprintframework/cert"
	"github.com/sprintframework/certpb"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	CertBucket    = "cert"
	CertBucketLen = len(CertBucket)
)

type implCertificateRepository struct {
	sync.Mutex
	Log       *zap.Logger     `inject`
	Store     store.DataStore `inject:"bean=config-store"`

	watchNum  atomic.Int64
	watchMap  sync.Map       // watchNum, configWatchContext

	shuttingDown  atomic.Bool

}

type zoneChangeEvent struct {
	zone string
	event string // UPDATE, DELETE
}

type zoneWatchContext struct {
	ctx       context.Context
	cancelFn  context.CancelFunc
	ch        chan<- zoneChangeEvent
}

func CertificateRepository() cert.CertificateRepository {
	return &implCertificateRepository{}
}

func (t *implCertificateRepository) Destroy() error {
	t.shuttingDown.Store(true)
	if t.watchNum.Load() > 0 {
		t.watchMap.Range(func(key, value interface{}) bool {
			if wc, ok := value.(*zoneWatchContext); ok {
				wc.cancelFn()
			}
			return true
		})
	}
	return nil
}

func (t *implCertificateRepository) SaveSelfSigner(self *certpb.SelfSigner) error {
	return t.Store.Set(context.Background()).ByKey("%s:self:%s", CertBucket, self.Name).Proto(self)
}

func (t *implCertificateRepository) FindSelfSigner(name string) (entry *certpb.SelfSigner, err error) {
	entry = new(certpb.SelfSigner)
	err = t.Store.Get(context.Background()).ByKey("%s:self:%s", CertBucket, name).ToProto(entry)
	return
}

func (t *implCertificateRepository) ListSelfSigners(prefix string, cb func(*certpb.SelfSigner) bool) error {
	return t.Store.Enumerate(context.Background()).ByPrefix("%s:self:%s", CertBucket, prefix).WithBatchSize(100).DoProto(func() proto.Message {
		return new(certpb.SelfSigner)
	}, func(entry *store.ProtoEntry) bool {
		if v, ok := entry.Value.(*certpb.SelfSigner); ok {
			return cb(v)
		}
		return true
	})
}

func (t *implCertificateRepository) DeleteSelfSigner(name string) error {
	return t.Store.Remove(context.Background()).ByKey("%s:self:%s", CertBucket, name).Do()
}

func (t *implCertificateRepository) SaveAccount(account *certpb.AcmeAccount) error {
	return t.Store.Set(context.Background()).ByKey("%s:acme:%s", CertBucket, account.Email).Proto(account)
}

func (t *implCertificateRepository) FindAccount(email string) (entry *certpb.AcmeAccount, err error) {
	entry = new(certpb.AcmeAccount)
	err = t.Store.Get(context.Background()).ByKey("%s:acme:%s", CertBucket, email).ToProto(entry)
	return
}

func (t *implCertificateRepository) ListAccounts(prefix string, cb func(*certpb.AcmeAccount) bool) error {
	return t.Store.Enumerate(context.Background()).ByPrefix("%s:acme:%s", CertBucket, prefix).WithBatchSize(100).DoProto(func() proto.Message {
		return new(certpb.AcmeAccount)
	}, func(entry *store.ProtoEntry) bool {
		if v, ok := entry.Value.(*certpb.AcmeAccount); ok {
			return cb(v)
		}
		return true
	})
}

func (t *implCertificateRepository) DeleteAccount(email string) error {
	return t.Store.Remove(context.Background()).ByKey("%s:acme:%s", CertBucket, email).Do()
}

func (t *implCertificateRepository) SaveZone(zone *certpb.Zone) error {
	err := t.Store.Set(context.Background()).ByKey("%s:zone:%s", CertBucket, zone.Zone).Proto(zone)
	if err == nil {
		t.notifyAll(zone.Zone, "UPDATE")
	}
	return err
}

func (t *implCertificateRepository) FindZone(zone string) (entry *certpb.Zone, err error) {
	entry = new(certpb.Zone)
	err = t.Store.Get(context.Background()).ByKey("%s:zone:%s", CertBucket, zone).ToProto(entry)
	return
}

func (t *implCertificateRepository) ListZones(prefix string, cb func(*certpb.Zone) bool) error {
	return t.Store.Enumerate(context.Background()).ByPrefix("%s:zone:%s", CertBucket, prefix).WithBatchSize(100).DoProto(func() proto.Message {
		return new(certpb.Zone)
	}, func(entry *store.ProtoEntry) bool {
		if v, ok := entry.Value.(*certpb.Zone); ok {
			return cb(v)
		}
		return true
	})
}

func (t *implCertificateRepository) DeleteZone(zone string) error {
	err := t.Store.Remove(context.Background()).ByKey("%s:zone:%s", CertBucket, zone).Do()
	if err == nil {
		t.notifyAll(zone, "DELETE")
	}
	return err
}

func (t *implCertificateRepository) notifyAll(zone, event string) {
	t.watchMap.Range(func(key, value interface{}) bool {
		if wc, ok := value.(*zoneWatchContext); ok {
			wc.ch <- zoneChangeEvent{zone, event}
		}
		return true
	})
}

func (t *implCertificateRepository) registerWatch(wc *zoneWatchContext) int64 {
	handle := t.watchNum.Inc()
	t.watchMap.Store(handle, wc)
	return handle
}

func (t *implCertificateRepository) unregisterWatch(handle int64) {
	t.watchMap.Delete(handle)
}

func (t *implCertificateRepository) Watch(ctx context.Context, cb func(zone, event string) bool) (cancel context.CancelFunc, err error) {

	ctx, cancel = context.WithCancel(ctx)
	ch := make(chan zoneChangeEvent)

	wc := &zoneWatchContext{
		ctx: ctx,
		cancelFn: cancel,
		ch: ch,
	}

	handle := t.registerWatch(wc)

	go func() {

		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					t.Log.Error("RecoverZoneWatcher", zap.Error(v))
				case string:
					t.Log.Error("RecoverZoneWatcher", zap.String("err", v))
				default:
					t.Log.Error("RecoverZoneWatcher", zap.String("err", fmt.Sprintf("%v", v)))
				}
			}
		}()

		defer func() {
			t.unregisterWatch(handle)
			close(ch)
		}()

		for {
			select {

			case <- ctx.Done():
				return

			case e := <- ch:
				if !cb(e.zone, e.event) {
					return
				}

			}
		}

	}()

	return
}

func (t *implCertificateRepository) Backend() store.DataStore {
	t.Lock()
	defer t.Unlock()
	return t.Store
}

func (t *implCertificateRepository) SetBackend(storage store.DataStore) {
	t.Lock()
	defer t.Unlock()
	t.Store = storage
}



