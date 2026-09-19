package api

import (
	"context"

	"github.com/gogi0001/family-tasks/internal/models"
)

type ctxKey int

const userCtxKey ctxKey = 0

// contextWithUser кладёт пользователя в контекст.
// Название с префиксом "context" — чтобы не конфликтовать с withUser-middleware.
func contextWithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func userFromCtx(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userCtxKey).(*models.User)
	return u, ok
}