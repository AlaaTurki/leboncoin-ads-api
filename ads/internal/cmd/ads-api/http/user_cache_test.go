package http

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	"sync"
	"testing"
	"time"
)

type stubUserClient struct {
	mu    sync.Mutex
	calls int
	user  usersclient.User
}

func (c *stubUserClient) GetUserByID(_ context.Context, _ model.UserID) (usersclient.User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.user, nil
}
func (c *stubUserClient) ListUsers(context.Context) ([]usersclient.User, error) {
	return []usersclient.User{c.user}, nil
}

func TestCachedUserClient(t *testing.T) {
	source := &stubUserClient{user: usersclient.User{UserID: "user-1", Nickname: "alice"}}
	cached := NewCachedUserClient(source, time.Minute)
	for range 2 {
		user, err := cached.GetUserByID(context.Background(), "user-1")
		require.NoError(t, err)
		require.Equal(t, "alice", user.Nickname)
	}
	require.Equal(t, 1, source.calls)
}
