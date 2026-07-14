package gateway

type Controller struct {
	routeTable RouteTable
}

func NewController(routeTable RouteTable) Controller {
	return Controller{
		routeTable: routeTable,
	}
}

func (c *Controller) AddRoutes(routes ...Route) Controller {
	c.routeTable = append(c.routeTable, routes...)

	return *c
}

func (r Controller) RouteTable() RouteTable {
	return r.routeTable
}
