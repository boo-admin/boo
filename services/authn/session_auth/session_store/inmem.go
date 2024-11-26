package session_store

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/goutils/tid"
	"github.com/boo-admin/boo/services/authn/session_auth"
)

const (
	CfgSessionRemoteApiKey       = "sessions.remote.api_key"
	CfgSessionInmemExpires       = "sessions.inmem.expires"
	CfgSessionInmemCheckInterval = "sessions.inmem.check_interval"
)

type onlineInfo struct {
	booclient.OnlineInfo

	t int64
}

func (o *onlineInfo) SwapUpdatedAt(t time.Time) time.Time {
	i64 := atomic.SwapInt64(&o.t, t.UnixNano())

	seconds := int64(i64) / int64(time.Second)
	nanoseconds := int64(i64) % int64(time.Second)
	return time.Unix(seconds, nanoseconds)
}

func (o *onlineInfo) SetUpdatedAt(t time.Time) {
	atomic.StoreInt64(&o.t, t.UnixNano())
}

func (o *onlineInfo) GetUpdatedAt() time.Time {
	i64 := atomic.LoadInt64(&o.t)

	seconds := int64(i64) / int64(time.Second)
	nanoseconds := int64(i64) % int64(time.Second)
	return time.Unix(seconds, nanoseconds)
}

func (o *onlineInfo) GetOnlineInfo() booclient.OnlineInfo {
	c := o.OnlineInfo
	c.UpdatedAt = o.GetUpdatedAt()
	return c
}

func CreateInmem(env *booclient.Environment) session_auth.Onlines {
	return &SessionManager{
		apiKey:   env.Config.StringWithDefault(CfgSessionRemoteApiKey, ""),
		filename: env.Fs.FromSession("boo_sessions.json"),
		expires:  time.Duration(env.Config.Int64WithDefault(CfgSessionInmemExpires, 0)) * time.Second,
		list:     map[string]*onlineInfo{},
	}
}

var _ session_auth.OnlineStore = &SessionManager{}

type SessionManager struct {
	filename string
	expires  time.Duration
	mu       sync.RWMutex
	list     map[string]*onlineInfo
	apiKey   string
}

func (mgr *SessionManager) Load(context.Context) error {
	if mgr.filename == "" {
		return nil
	}

	bs, err := ioutil.ReadFile(mgr.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(bs) == 0 {
		return nil
	}

	var list []booclient.OnlineInfo
	err = json.Unmarshal(bs, &list)
	if err != nil {
		return err
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	for idx := range list {
		mgr.list[list[idx].UUID] = &onlineInfo{
			OnlineInfo: list[idx],
			t:          list[idx].UpdatedAt.UnixNano(),
		}
	}
	return nil
}

func (mgr *SessionManager) Store(ctx context.Context) error {
	if mgr.filename == "" {
		return nil
	}

	list, err := mgr.List(ctx)
	if err != nil {
		return err
	}

	bs, err := json.Marshal(list)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(mgr.filename), 0777); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	err = ioutil.WriteFile(mgr.filename, bs, 0666)
	return err
}

func (mgr *SessionManager) Count(ctx context.Context) (int64, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	mgr.cleanExpiredInReadlock(ctx)

	return int64(len(mgr.list)), nil
}

func (mgr *SessionManager) UpdateNow(ctx context.Context, uuid, apiKey string) error {
	if apiKey != mgr.apiKey {
		return errors.New("session api key is invalid")
	}

	mgr.mu.RLock()
	s, ok := mgr.list[uuid]
	mgr.mu.RUnlock()
	if !ok {
		return session_auth.ErrSessionNotExists
	}

	now := time.Now()
	old := s.SwapUpdatedAt(now)

	if mgr.expires > 0 {
		if now.Sub(old) > mgr.expires {
			return session_auth.ErrSessionExpired
		}
	}
	return nil
}

func (mgr *SessionManager) GetBySessionID(ctx context.Context, uuid string) (*booclient.OnlineInfo, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	a := mgr.list[uuid]
	if a == nil {
		return nil, nil
	}
	copyed := a.GetOnlineInfo()
	return &copyed, nil
}

func (mgr *SessionManager) List(ctx context.Context) ([]booclient.OnlineInfo, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	mgr.cleanExpiredInReadlock(ctx)

	var results = make([]booclient.OnlineInfo, 0, len(mgr.list))
	for _, s := range mgr.list {
		results = append(results, s.GetOnlineInfo())
	}
	return results, nil
}

func (mgr *SessionManager) Login(ctx context.Context, username, loginAddress, apiKey string) (string, error) {
	if apiKey != mgr.apiKey {
		return "", errors.New("session api key is invalid")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	var old *booclient.OnlineInfo

	for _, s := range mgr.list {
		if s.Username == username && s.Address == loginAddress {
			o := s.GetOnlineInfo()
			old = &o
			break
		}
	}
	if old != nil {
		return old.UUID, nil
	}

	uuid := tid.GenerateID()
	mgr.list[uuid] = &onlineInfo{
		OnlineInfo: booclient.OnlineInfo{
			UUID:      uuid,
			Username:  username,
			Address:   loginAddress,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		t: time.Now().UnixNano(),
	}
	return uuid, nil
}

func (mgr *SessionManager) LogoutByUsername(ctx context.Context, username string) error {
	idlist := func() []string {
		mgr.mu.RLock()
		defer mgr.mu.RUnlock()

		var idlist []string
		for id, s := range mgr.list {
			if s.Username == username {
				idlist = append(idlist, id)
			}
		}
		return idlist
	}()
	if len(idlist) == 0 {
		return nil
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	for _, id := range idlist {
		delete(mgr.list, id)
	}
	return nil
}

func (mgr *SessionManager) LogoutBySessionID(ctx context.Context, id string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	delete(mgr.list, id)
	return nil
}

func (mgr *SessionManager) cleanExpiredInReadlock(ctx context.Context) {
	if mgr.expires <= 0 {
		return
	}
	var idlist []string
	now := time.Now()
	for id, s := range mgr.list {
		updatedAt := s.GetUpdatedAt()
		if now.Sub(updatedAt) > mgr.expires {
			idlist = append(idlist, id)
		}
	}
	if len(idlist) == 0 {
		return
	}

	mgr.mu.RUnlock()
	mgr.mu.Lock()

	defer func() {
		mgr.mu.Unlock()
		mgr.mu.RLock()
	}()

	for _, id := range idlist {
		delete(mgr.list, id)
	}
}

func (mgr *SessionManager) IsOnlineExists(ctx context.Context, username, loginAddress string) error {
	// 判断用户是不是已经在其它主机上登录
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	mgr.cleanExpiredInReadlock(ctx)

	var onlineList = make([]booclient.OnlineInfo, 0, 4)

	for _, s := range mgr.list {
		if s.Username == username {
			if s.Address == loginAddress {
				return nil
			}
			onlineList = append(onlineList, s.GetOnlineInfo())
		}
	}

	if len(onlineList) > 0 {
		return &session_auth.ErrOnline{OnlineList: onlineList}
	}
	return nil
}

func (mgr *SessionManager) DeleteExpired(ctx context.Context) error {
	if mgr.expires <= 0 {
		return nil
	}

	idlist := func() []string {
		now := time.Now()

		mgr.mu.RLock()
		defer mgr.mu.RUnlock()

		var idlist []string
		for id, s := range mgr.list {
			if now.Sub(s.GetUpdatedAt()) > mgr.expires {
				idlist = append(idlist, id)
			}
		}
		return idlist
	}()

	if len(idlist) == 0 {
		return nil
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	for _, id := range idlist {
		delete(mgr.list, id)
	}
	return nil
}
