package sites

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// Sites are mostly relational behaviour, so these run against a real database.
// CI has no Postgres service, so they skip unless TEST_DATABASE_URL is set:
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/sites/
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return pool
}

// seedCustomer creates a throwaway customer and removes it (and, by cascade,
// its sites) afterwards.
func seedCustomer(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status)
		 VALUES ($1, '', '', '0800000000', 'Test address', 'other', 'Test area', 'active')
		 RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, id)
	})
	return id
}

func TestDBCustomerCanHaveManySites(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	svc := NewService(repo)
	ctx := context.Background()

	customerID := seedCustomer(t, pool, "Branches Co "+time.Now().Format("150405.000000"))

	for _, name := range []string{"Sukhumvit Branch", "Silom Branch", "Asoke Branch"} {
		if _, err := svc.Create(ctx, CreateSiteRequest{
			CustomerID:   customerID,
			Name:         name,
			Address:      name + " Road",
			PropertyType: string(PropertyRestaurant),
		}); err != nil {
			t.Fatalf("create site %q: %v", name, err)
		}
	}

	items, total, err := svc.List(ctx, Filters{CustomerID: customerID}, pagination.Params{Limit: 50})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("got %d sites (total %d), want 3", len(items), total)
	}
	for _, s := range items {
		if s.CustomerID != customerID {
			t.Fatalf("site %d belongs to customer %d, want %d", s.ID, s.CustomerID, customerID)
		}
	}
}

func TestDBEnsureOwnedBy(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	svc := NewService(repo)
	ctx := context.Background()

	stamp := time.Now().Format("150405.000000")
	customerA := seedCustomer(t, pool, "Owner A "+stamp)
	customerB := seedCustomer(t, pool, "Owner B "+stamp)

	siteA, err := svc.Create(ctx, CreateSiteRequest{CustomerID: customerA, Name: "A site"})
	if err != nil {
		t.Fatalf("create site for A: %v", err)
	}

	if err := svc.EnsureOwnedBy(ctx, siteA.ID, customerA); err != nil {
		t.Fatalf("owner rejected for its own site: %v", err)
	}

	err = svc.EnsureOwnedBy(ctx, siteA.ID, customerB)
	if err == nil {
		t.Fatal("customer B was allowed to use customer A's site")
	}
	var apiErr *response.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 422 {
		t.Fatalf("got %v, want a 422 API error", err)
	}

	if err := svc.EnsureOwnedBy(ctx, 0, customerA); err == nil {
		t.Fatal("expected a missing site to be rejected")
	}
}

// Deleting a customer must take their sites with them rather than leaving
// orphans behind.
func TestDBSitesCascadeWithCustomer(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	svc := NewService(repo)
	ctx := context.Background()

	customerID := seedCustomer(t, pool, "Cascade Co "+time.Now().Format("150405.000000"))
	site, err := svc.Create(ctx, CreateSiteRequest{CustomerID: customerID, Name: "Doomed site"})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, customerID); err != nil {
		t.Fatalf("delete customer: %v", err)
	}
	if _, err := svc.Get(ctx, site.ID); err == nil {
		t.Fatal("site outlived its customer")
	}
}

func TestDBCreateRejectsUnknownCustomer(t *testing.T) {
	pool := testPool(t)
	svc := NewService(NewRepository(pool))
	if _, err := svc.Create(context.Background(), CreateSiteRequest{
		CustomerID: 9_999_999,
		Name:       "Nowhere",
	}); err == nil {
		t.Fatal("expected a site for a nonexistent customer to be rejected")
	}
}
