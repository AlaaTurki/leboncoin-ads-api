package http

import (
	"context"
	"errors"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
	"sync"
	"time"
)

type cachedUser struct {
	user      usersclient.User
	err       error
	expiresAt time.Time
}
type userLookup struct {
	done chan struct{}
	user usersclient.User
	err  error
}

// cachedUserClient limits identity-service traffic on the read-heavy ad endpoint.
type cachedUserClient struct {
	client      usersclient.UserClient
	ttl         time.Duration
	mu          sync.Mutex
	users       map[model.UserID]cachedUser
	inFlight    map[model.UserID]*userLookup
	nextCleanup time.Time
}

func NewCachedUserClient(client usersclient.UserClient, ttl time.Duration) usersclient.UserClient {
	return &cachedUserClient{client: client, ttl: ttl, users: make(map[model.UserID]cachedUser), inFlight: make(map[model.UserID]*userLookup)}
}

func (c *cachedUserClient) GetUserByID(ctx context.Context, id model.UserID) (usersclient.User, error) {
	now := time.Now()
	c.mu.Lock()
	if !now.Before(c.nextCleanup) {
		for key, entry := range c.users {
			if !now.Before(entry.expiresAt) {
				delete(c.users, key)
			}
		}
		c.nextCleanup = now.Add(c.ttl)
	}
	if entry, ok := c.users[id]; ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.user, entry.err
	}
	if pending, ok := c.inFlight[id]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return usersclient.User{}, ctx.Err()
		case <-pending.done:
			return pending.user, pending.err
		}
	}
	pending := &userLookup{done: make(chan struct{})}
	c.inFlight[id] = pending
	c.mu.Unlock()

	pending.user, pending.err = c.client.GetUserByID(ctx, id)
	c.mu.Lock()
	if c.ttl > 0 && (pending.err == nil || errors.Is(pending.err, usersapi.ErrNotFound)) {
		c.users[id] = cachedUser{user: pending.user, err: pending.err, expiresAt: time.Now().Add(c.ttl)}
	}
	delete(c.inFlight, id)
	close(pending.done)
	c.mu.Unlock()
	return pending.user, pending.err
}

func (c *cachedUserClient) ListUsers(ctx context.Context) ([]usersclient.User, error) {
	return c.client.ListUsers(ctx)
}
