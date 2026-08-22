package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shoenig/test/must"
)

func Test_discoverServices(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	for name, methods := range map[string]string{testService: readAndList, otherService: allMethods} {
		spec, err := newServiceSpec(testModule, name, methods, false)
		must.NoError(t, err)

		_, err = generateService(spec, root)
		must.NoError(t, err)
	}

	// a directory that is not a service is skipped
	err := os.MkdirAll(filepath.Join(root, "services", "notaservice"), dirPerm)
	must.NoError(t, err)

	services, err := discoverServices(root, testModule)
	must.NoError(t, err)
	must.Eq(t, []string{"category", serviceLower}, serviceLowers(services))

	order, err := findService(services, testService)
	must.NoError(t, err)
	must.Eq(t, serviceLower, order.Package)
	must.Eq(t, testModule+"/services/order/order-client", order.Import)
	must.Eq(t, testService, order.Entity)
	must.Eq(t, serviceRPC{
		Key:      methodRead,
		Name:     "GetOrderRPC",
		Request:  "GetOrderRequest",
		Response: testService,
	}, order.RPCs[methodRead])
	must.MapContainsKeys(t, order.RPCs, []string{methodRead, methodList})
	must.MapNotContainsKeys(t, order.RPCs, []string{methodCreate, methodUpdate, methodDelete})

	category, err := findService(services, "category")
	must.NoError(t, err)
	must.Eq(t, 5, len(category.RPCs))
	must.Eq(t, "SelectCategoriesRPC", category.RPCs[methodList].Name)
	must.Eq(t, "DeletedCategory", category.RPCs[methodDelete].Response)

	_, err = findService(services, "invoice")
	must.ErrorIs(t, err, errUnknownSvc)
}

func Test_discoverServices_EmptyProject(t *testing.T) {
	t.Parallel()

	services, err := discoverServices(t.TempDir(), testModule)
	must.NoError(t, err)
	must.SliceEmpty(t, services)
}

func serviceLowers(services []serviceInfo) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Lower)
	}

	return names
}
