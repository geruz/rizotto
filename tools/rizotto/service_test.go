package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/shoenig/test/must"
	"gopkg.in/yaml.v3"
)

const (
	testModule  = "github.com/acme/shop"
	readAndList = "read,list"
)

func Test_normalizeMethods(t *testing.T) {
	t.Parallel()

	every := "create,read,list,update,delete"
	valid := map[string]string{
		allMethods:              every,
		"crud":                  every,
		"delete, create":        "create,delete",
		"c,r,l,u,d":             every,
		"get list":              readAndList,
		" Read , READ , List  ": readAndList,
	}

	for raw, expected := range valid {
		normalized, err := normalizeMethods(raw)
		must.NoError(t, err, must.Sprintf("methods %q", raw))
		must.Eq(t, expected, normalized, must.Sprintf("methods %q", raw))
	}

	for _, raw := range []string{"", "everything", "create,drop"} {
		_, err := normalizeMethods(raw)
		must.Error(t, err, must.Sprintf("methods %q must be rejected", raw))
	}
}

func Test_newServiceSpec_RejectsAPluralName(t *testing.T) {
	t.Parallel()

	_, err := newServiceSpec(testModule, pluralName, allMethods, true)
	must.ErrorIs(t, err, errPluralName)
}

func Test_newServiceSpec(t *testing.T) {
	t.Parallel()

	spec, err := newServiceSpec(testModule, serviceLower, readAndList, true)
	must.NoError(t, err)
	must.Eq(t, testService, spec.Name)
	must.Eq(t, "Orders", spec.NamePlural)
	must.Eq(t, serviceLower, spec.Lower)
	must.Eq(t, "orders", spec.LowerPlural)
	must.Eq(t, "ordersrv", spec.Package)
	must.Eq(t, "services/"+serviceLower, spec.Dir())
	must.Eq(t, "ordersrv.NewOrderService(pgPool)", spec.Constructor())
	must.True(t, spec.NeedsID)
	must.True(t, spec.NeedsModel)
	must.False(t, spec.Create)

	must.Eq(t, []serviceMethod{
		{
			Name:     "GetOrderRPC",
			Request:  "GetOrderRequest",
			Response: "Order",
			Address:  "http://order-service/order/get",
		},
		{
			Name:     "SelectOrdersRPC",
			Request:  "SelectOrdersRequest",
			Response: "OrderList",
			Address:  "http://order-service/order/select",
		},
	}, spec.Methods)

	memory, err := newServiceSpec(testModule, testService, "create", false)
	must.NoError(t, err)
	must.Eq(t, "ordersrv.NewOrderService()", memory.Constructor())
	must.False(t, memory.NeedsID)

	_, err = newServiceSpec(testModule, "my-order", allMethods, false)
	must.Error(t, err)

	_, err = newServiceSpec(testModule, testService, "nothing", false)
	must.Error(t, err)
}

func Test_generateService_WithDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	spec, err := newServiceSpec(testModule, testService, allMethods, true)
	must.NoError(t, err)

	files, err := generateService(spec, root)
	must.NoError(t, err)

	mustHaveFiles(t, root, files,
		"services/order/order-client/bind-gen.go",
		"services/order/order-client/client.go",
		"services/order/order-service.go",
		"services/order/repository/db/db.go",
		"services/order/repository/db/models.go",
		"services/order/repository/db/order-queries.sql.go",
		"services/order/repository/migrations/000001_create_orders.sql",
		"services/order/repository/order-repository.go",
		"services/order/repository/sql/order-queries.sql",
		"services/order/repository/sql/schema.sql",
		"sqlc.yaml",
	)

	queries := readFile(t, root, "services/order/repository/sql/order-queries.sql")
	for _, name := range []string{"CreateOrder", "GetOrder", "SelectOrders", "UpdateOrder", "DeleteOrder"} {
		must.StrContains(t, queries, "-- name: "+name)
	}

	must.StrContains(t, readFile(t, root, "services/order/repository/sql/schema.sql"), "CREATE TABLE public.orders")
	must.StrContains(t, readFile(t, root, "services/order/order-service.go"), "repository.NewOrderRepository(pool)")
	must.StrContains(t, readFile(t, root, "services/order/order-client/client.go"),
		"// bind-method: http://order-service/order/create")
}

// Test_generateService_StoresIntegersAs64Bit pins the column conventions the
// repository skill documents: BIGSERIAL keys typed as int64 all the way up, so
// no service needs a math.MaxInt32 guard to reach its own table.
func Test_generateService_StoresIntegersAs64Bit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	spec, err := newServiceSpec(testModule, testService, allMethods, true)
	must.NoError(t, err)

	_, err = generateService(spec, root)
	must.NoError(t, err)

	for _, file := range []string{
		"services/order/repository/sql/schema.sql",
		"services/order/repository/migrations/000001_create_orders.sql",
	} {
		must.StrContains(t, readFile(t, root, file), "id BIGSERIAL PRIMARY KEY")
	}

	must.StrContains(t, readFile(t, root, "services/order/repository/db/models.go"), "ID        int64")
	must.StrContains(t, readFile(t, root, "services/order/repository/order-repository.go"),
		"GetOrderByID(ctx context.Context, id int64)")

	// LIMIT/OFFSET have no column to take their type from, so sqlc types a bare
	// placeholder as int32; the cast is what keeps the pair int64.
	must.StrContains(t, readFile(t, root, "services/order/repository/sql/order-queries.sql"),
		"LIMIT sqlc.arg('limit')::bigint OFFSET sqlc.arg('offset')::bigint")

	service := readFile(t, root, "services/order/order-service.go")
	must.StrContains(t, service, "func toOrderID(id int) (int64, bb.ServiceError)")
	must.StrNotContains(t, service, "math.MaxInt32")
	must.StrNotContains(t, service, "int32(")
}

// Test_generateService_TimestampsHaveNoTimeZone pins the other half of the
// convention: TIMESTAMP, so Postgres never converts on read.
func Test_generateService_TimestampsHaveNoTimeZone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	spec, err := newServiceSpec(testModule, testService, allMethods, true)
	must.NoError(t, err)

	_, err = generateService(spec, root)
	must.NoError(t, err)

	for _, file := range []string{
		"services/order/repository/sql/schema.sql",
		"services/order/repository/migrations/000001_create_orders.sql",
	} {
		schema := readFile(t, root, file)
		must.StrContains(t, schema, "created_at TIMESTAMP NOT NULL DEFAULT NOW()")
		must.StrContains(t, schema, "updated_at TIMESTAMP NOT NULL DEFAULT NOW()")
		must.StrNotContains(t, schema, "TIMESTAMPTZ")
	}

	must.StrContains(t, readFile(t, root, "services/order/repository/db/models.go"), "pgtype.Timestamp")
}

func Test_generateService_WithoutDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	spec, err := newServiceSpec(testModule, otherService, "create,read,list", false)
	must.NoError(t, err)

	files, err := generateService(spec, root)
	must.NoError(t, err)

	must.Eq(t, []string{
		"services/category/category-client/bind-gen.go",
		"services/category/category-client/client.go",
		"services/category/category-service.go",
	}, sorted(files))

	mustHaveFiles(t, root, files)

	service := readFile(t, root, "services/category/category-service.go")
	must.StrContains(t, service, "categories []category.Category")
	must.StrContains(t, service, "func NewCategoryService() *CategoryService")
	must.StrNotContains(t, service, "/repository")

	client := readFile(t, root, "services/category/category-client/client.go")
	must.StrContains(t, client, "SelectCategoriesRequest")
	must.StrNotContains(t, client, "UpdateCategoryRequest")
	must.StrNotContains(t, client, "DeleteCategoryRequest")

	_, err = os.Stat(filepath.Join(root, sqlcConfigFile))
	must.True(t, os.IsNotExist(err), must.Sprint("a service without a database must not create sqlc.yaml"))
}

func Test_addSqlcEntry_AppendsAndSkipsDuplicates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	first, err := newServiceSpec(testModule, testService, "read", true)
	must.NoError(t, err)

	second, err := newServiceSpec(testModule, "Invoice", "read", true)
	must.NoError(t, err)

	for _, spec := range []serviceSpec{first, second} {
		changed, addErr := addSqlcEntry(root, spec)
		must.NoError(t, addErr)
		must.True(t, changed)
	}

	changed, err := addSqlcEntry(root, first)
	must.NoError(t, err)
	must.False(t, changed, must.Sprint("a service must be registered in sqlc.yaml only once"))

	var config struct {
		Version string `yaml:"version"`
		SQL     []struct {
			Queries string `yaml:"queries"`
		} `yaml:"sql"`
	}

	err = yaml.Unmarshal([]byte(readFile(t, root, sqlcConfigFile)), &config)
	must.NoError(t, err)
	must.Eq(t, "2", config.Version)
	must.Eq(t, 2, len(config.SQL))
	must.Eq(t, "./services/order/repository/sql/order-queries.sql", config.SQL[0].Queries)
	must.Eq(t, "./services/invoice/repository/sql/invoice-queries.sql", config.SQL[1].Queries)
}

func Test_addSqlcEntry_KeepsTheRestOfTheFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	original := `# keep me
version: "2"
sql:
    - engine: "postgresql"
      queries: "./services/item/repository/sql/item-queries.sql"
      schema:
          - "./services/item/repository/sql/schema.sql"
      gen:
          go:
              sql_package: pgx/v5
              out: ./services/item/repository/db
`

	err := os.WriteFile(filepath.Join(root, sqlcConfigFile), []byte(original), filePerm)
	must.NoError(t, err)

	spec, err := newServiceSpec(testModule, testService, "read", true)
	must.NoError(t, err)

	_, err = addSqlcEntry(root, spec)
	must.NoError(t, err)

	updated := readFile(t, root, sqlcConfigFile)
	must.StrContains(t, updated, "# keep me")
	must.StrContains(t, updated, "./services/item/repository/sql/item-queries.sql")
	must.StrContains(t, updated, "./services/order/repository/sql/order-queries.sql")
}

func sorted(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)

	return out
}
