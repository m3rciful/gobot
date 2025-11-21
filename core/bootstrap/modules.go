package bootstrap

import "context"

// Storage represents shared infrastructure passed to optional modules.
type Storage interface{}

// Seeder loads reference data into a storage implementation.
type Seeder interface {
	Seed(ctx context.Context, storage Storage) error
}

// SeederFunc adapts a bare function to the Seeder interface.
type SeederFunc func(ctx context.Context, storage Storage) error

// Seed executes the underlying function.
func (f SeederFunc) Seed(ctx context.Context, storage Storage) error {
	return f(ctx, storage)
}

// ServiceProvider wires application services using configuration and storage.
type ServiceProvider interface {
	Provide(ctx context.Context, cfg interface{}, storage Storage) (interface{}, error)
}

// ServiceProviderFunc adapts a function to the ServiceProvider interface.
type ServiceProviderFunc func(ctx context.Context, cfg interface{}, storage Storage) (interface{}, error)

// Provide executes the underlying function.
func (f ServiceProviderFunc) Provide(ctx context.Context, cfg interface{}, storage Storage) (interface{}, error) {
	return f(ctx, cfg, storage)
}

// TypedServiceProvider allows callers to avoid manual type assertions.
type TypedServiceProvider[T any] interface {
	ServiceProvider
	ProvideTyped(ctx context.Context, cfg interface{}, storage Storage) (T, error)
}

// TypedServiceProviderFunc adapts a typed function to both typed and untyped provider interfaces.
type TypedServiceProviderFunc[T any] func(ctx context.Context, cfg interface{}, storage Storage) (T, error)

// Provide satisfies the ServiceProvider interface.
func (f TypedServiceProviderFunc[T]) Provide(ctx context.Context, cfg interface{}, storage Storage) (interface{}, error) {
	return f(ctx, cfg, storage)
}

// ProvideTyped exposes the typed return value without casting.
func (f TypedServiceProviderFunc[T]) ProvideTyped(ctx context.Context, cfg interface{}, storage Storage) (T, error) {
	return f(ctx, cfg, storage)
}

// Modules groups optional bootstrapping hooks for seeding and service initialization.
type Modules struct {
	Seeders  []Seeder
	Services ServiceProvider
}
