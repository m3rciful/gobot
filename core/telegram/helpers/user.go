package helpers

import "context"

// CurrentUser resolves a Telegram user ID to a domain entity via a service that
// implements GetUserByTelegramID. The generic type T allows different projects
// to supply their own user model.
func CurrentUser[T any](
	ctx context.Context,
	service interface {
		GetUserByTelegramID(context.Context, int64) (T, error)
	},
	tgID int64,
) (T, error) {
	var zero T
	if service == nil {
		return zero, nil
	}
	return service.GetUserByTelegramID(ctx, tgID)
}
