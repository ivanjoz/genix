package datafunctions

// Functions is the catalog genix-bot searches and calls.
var Functions = Catalog(salesFunctions)

// salesFunctions read the four sale datasets (sales.SaleFrameSQLFrames). Their descriptions carry the
// traps of the data: annulled sales add nothing; debt_amount is what the period's sales still owe, not
// a client's whole debt; quantity is each product's own unit, so it shows only per product (rule Q);
// no dataset has client and site together (rule F); a product in two categories counts in both; there
// is no number of sales nor a ticket per sale.
var salesFunctions = []Function{
	// ─── Totals and time ─────────────────────────────────────────────────────────
	{
		Name: "sales_total", Domain: "sales", Access: salesAccess,
		Title: "Total de ventas", Gist: "monto vendido en un periodo, con filtros opcionales",
		Description: "El monto total vendido en el periodo, en soles y sin ventas anuladas; con productos da también sus unidades. " +
			"No da el número de ventas ni el ticket promedio, y no combina clientes con sedes.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"cuánto vendí", "total de ventas", "monto vendido en el periodo", "cuánto se vendió en total",
			"suma de todas las ventas", "cuanto vendimos", "cuánto vendí de un producto", "cuánto le vendí a un cliente", "total vendido en una sede",
			"cuánto he vendido", "cuánto llevo vendido", "a cuánto suman las ventas"},
		Message: "Total vendido {periodo}{filtros}: {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.total("amount") },
	},
	{
		Name: "sales_by_day", Domain: "sales", Access: salesAccess,
		Title: "Ventas por día", Gist: "línea del monto vendido cada día del periodo",
		Description: "Una línea con el monto vendido cada día del periodo, en soles y sin ventas anuladas; los días sin ventas quedan en cero. " +
			"No da el número de ventas por día ni combina clientes con sedes.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"ventas diarias", "cuánto vendí cada día", "evolución diaria de las ventas", "gráfico de ventas por día",
			"ventas día por día", "tendencia diaria de ventas", "cómo fueron las ventas cada día", "ventas por fecha", "venta diaria de una sede",
			"ventas de cada día", "total vendido por día", "ventas diarias de un producto",
			"monto diario del periodo", "cuánto se vendió en cada fecha"},
		Message: "Ventas por día {periodo}{filtros}: {valor} en total.",
		run:     func(call *functionCall) (shownResult, error) { return call.timeline("date", "amount", "line") },
	},
	{
		Name: "sales_by_week", Domain: "sales", Access: salesAccess,
		Title: "Ventas por semana", Gist: "barras del monto vendido en cada semana, de lunes a domingo",
		Description: "Barras con el monto vendido en cada semana del periodo (de lunes a domingo), en soles y sin ventas anuladas. " +
			"No da unidades, porque cada producto tiene su propia unidad, ni el número de ventas.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"ventas semanales", "cuánto vendí cada semana", "ventas por semana", "evolución semanal de las ventas",
			"gráfico semanal de ventas", "semana a semana cuánto vendimos", "total de cada semana", "ventas semanales de una sede",
			"ventas de cada semana", "total vendido por semana", "semanas con más ventas",
			"tendencia semanal de las ventas", "ventas totales por semana"},
		Message: "Ventas por semana {periodo}{filtros}: {valor} en total.",
		run:     func(call *functionCall) (shownResult, error) { return call.timeline("WEEK(date)", "amount", "bar") },
	},
	{
		Name: "sales_by_month", Domain: "sales", Access: salesAccess,
		Title: "Ventas por mes", Gist: "barras del monto vendido en cada mes del periodo",
		Description: "Barras con el monto vendido en cada mes del periodo, en soles y sin ventas anuladas. " +
			"No da unidades, porque cada producto tiene su propia unidad, ni el número de ventas.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"ventas mensuales", "cuánto vendí cada mes", "ventas por mes del año", "evolución mensual de las ventas",
			"gráfico de ventas mensuales", "mes a mes cuánto se vendió", "total de cada mes", "ventas mensuales de un cliente"},
		Message: "Ventas por mes {periodo}{filtros}: {valor} en total.",
		run:     func(call *functionCall) (shownResult, error) { return call.timeline("MONTH(date)", "amount", "bar") },
	},
	{
		Name: "average_daily_sales", Domain: "sales", Access: salesAccess,
		Title: "Venta promedio diaria", Gist: "monto vendido dividido entre todos los días, con o sin ventas",
		Description: "El promedio diario: el monto vendido dividido entre todos los días del periodo, contando en cero los días sin ventas; en soles y sin ventas anuladas. " +
			"No es el ticket promedio por venta: no hay número de ventas.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"cuánto vendo en promedio al día", "promedio diario de ventas", "venta media por día", "en promedio cuánto vendemos diario",
			"promedio de ventas por día", "cuánto se vende al día en promedio", "media diaria de ventas", "venta diaria promedio de un producto"},
		Message: "Venta promedio diaria {periodo}{filtros}: {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.averageDaily() },
	},
	{
		Name: "sales_by_weekday", Domain: "sales", Access: salesAccess,
		Title: "Ventas promedio por día de la semana", Gist: "promedio vendido cada lunes, martes y demás días de la semana",
		Description: "Barras con el promedio vendido cada día de la semana, dividiendo entre todos esos días del periodo, también los días sin ventas; " +
			"en soles y sin ventas anuladas. No da el número de ventas.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"qué día de la semana vendo más", "ventas por día de la semana", "cuál es el mejor día para vender",
			"promedio de ventas de cada día de la semana", "vendo más los fines de semana", "días de la semana con más ventas",
			"ventas de lunes a domingo", "qué día se vende menos"},
		Message: "Venta promedio por día de la semana {periodo}{filtros}.",
		run:     func(call *functionCall) (shownResult, error) { return call.weekdayAverages() },
	},

	// ─── Breakdowns and rankings ─────────────────────────────────────────────────
	{
		Name: "sales_by_product", Domain: "sales", Access: salesAccess,
		Title: "Ventas por producto", Gist: "ranking de productos por monto o por unidades vendidas",
		Description: "Ranking de productos con el monto en soles y las unidades vendidas de cada uno, sin ventas anuladas; ordena por monto o por unidades, " +
			"y las unidades son las de cada producto. No combina clientes con sedes.",
		Arguments: arguments("period?", "clients?", "categories?", "sites?", "top?", "rank_by?"),
		Examples: []string{"productos más vendidos", "top de productos", "ranking de productos por ventas", "qué producto se vende más",
			"productos con más unidades vendidas", "qué productos le vendí a un cliente", "productos más vendidos de una categoría",
			"cuánto vendí de cada producto", "productos estrella",
			"monto vendido por producto"},
		Message: "Productos más vendidos {periodo}{filtros}, ordenados {orden}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("product_id", []string{"amount", "quantity"}, call.rankBy, call.top)
		},
	},
	{
		Name: "sales_by_client", Domain: "sales", Access: salesAccess,
		Title: "Ventas por cliente", Gist: "ranking de clientes por monto comprado",
		Description: "Quién compró y cuánto: ranking de clientes por el monto que compraron en el periodo, en soles y sin ventas anuladas; las ventas sin cliente salen como «(ninguno)». " +
			"No hay datos por cliente y sede a la vez, ni número de compras.",
		Arguments: arguments("period?", "products?", "categories?", "top?"),
		Examples: []string{"mejores clientes", "clientes que más compran", "top de clientes", "ranking de clientes por compras", "quién me compra más",
			"clientes que más compraron un producto", "a quién le vendí más", "principales clientes por monto",
			"a qué clientes les vendí un producto", "compradores de un producto", "monto comprado por cada cliente", "ranking de compradores",
			"clientes que más gastan", "clientes con más compras en soles"},
		Message: "Clientes que más compraron {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("client_id", []string{"amount"}, "amount", call.top)
		},
	},
	{
		Name: "sales_by_category", Domain: "sales", Access: salesAccess,
		Title: "Ventas por categoría", Gist: "ranking de categorías de producto por monto vendido",
		Description: "Ranking de categorías de producto por monto vendido, en soles y sin ventas anuladas. " +
			"Un producto en dos categorías cuenta en ambas, así que las categorías suman más que el total de ventas; no combina clientes con sedes.",
		Arguments: arguments("period?", "clients?", "sites?", "top?"),
		Examples: []string{"categorías más vendidas", "ventas por categoría", "qué categoría vende más", "ranking de categorías",
			"cuánto vendí por tipo de producto", "líneas de producto más vendidas", "ventas por familia de productos", "categorías que más compra un cliente"},
		Message: "Categorías más vendidas {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.ranking("category_id", []string{"amount"}, "amount", call.top)
			shown.note = categoryOverlapNote
			return shown, err
		},
	},
	{
		Name: "sales_by_site", Domain: "sales", Access: salesAccess,
		Title: "Ventas por sede", Gist: "barras del monto vendido en cada sede o local",
		Description: "Barras con el monto vendido en cada sede, en soles y sin ventas anuladas; las ventas sin almacén salen como «(ninguno)». " +
			"No hay datos por sede y cliente a la vez.",
		Arguments: arguments("period?", "products?", "categories?"),
		Examples: []string{"ventas por sede", "qué sede vende más", "ventas de cada local", "cuánto vendió cada tienda", "ranking de sedes",
			"ventas por sucursal", "comparar las sedes en ventas", "en qué local se vende más un producto",
			"monto de ventas de cada sede", "total vendido por sede",
			"ventas totales de cada sede"},
		Message: "Ventas por sede {periodo}{filtros}: {valor} en total.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.ranking("site_id", []string{"amount"}, "amount", MaxTableRows)
			return asBars(shown, "site_id", "amount"), err
		},
	},
	{
		Name: "sales_share_by_product", Domain: "sales", Access: salesAccess,
		Title: "Participación de productos en las ventas", Gist: "porcentaje del total vendido de cada producto, con el resto en «Otros»",
		Description: "Qué porcentaje del monto total vendido aporta cada producto principal, con una fila «Otros» para el resto; en soles y sin ventas anuladas.",
		Arguments:   arguments("period?", "sites?", "top?"),
		Examples: []string{"porcentaje de ventas por producto", "participación de cada producto en las ventas", "qué parte de las ventas es de cada producto",
			"cuánto pesa cada producto en las ventas", "mix de ventas por producto", "productos que concentran las ventas",
			"porcentaje del total que representa cada producto", "share de ventas por producto"},
		Message: "Participación de los productos en las ventas {periodo}{filtros}, de un total de {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.share("product_id", call.top) },
	},
	{
		Name: "sales_share_by_site", Domain: "sales", Access: salesAccess,
		Title: "Participación de sedes en las ventas", Gist: "porcentaje del total vendido que aporta cada sede",
		Description: "Qué porcentaje del monto total vendido aporta cada sede, en soles y sin ventas anuladas; las ventas sin almacén salen como «(ninguno)».",
		Arguments:   arguments("period?", "products?", "categories?"),
		Examples: []string{"porcentaje de ventas por sede", "participación de cada sede", "qué parte de las ventas hace cada local",
			"cuánto aporta cada tienda al total", "peso de cada sucursal en las ventas", "distribución de las ventas entre sedes",
			"porcentaje que vende cada local", "share de ventas por sede",
			"cuánto representa cada sede del total", "porcentaje del total de cada sucursal"},
		Message: "Participación de las sedes en las ventas {periodo}{filtros}, de un total de {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.share("site_id", MaxTableRows) },
	},

	// ─── Series: one line per record, at most 12 ─────────────────────────────────
	{
		Name: "sales_by_product_and_day", Domain: "sales", Access: salesAccess,
		Title: "Ventas diarias por producto", Gist: "una línea por producto con su monto vendido cada día",
		Description: "Una línea por cada producto pedido, hasta 12, con el monto vendido cada día, en soles y sin ventas anuladas; sirve para comparar productos día a día. " +
			"No combina clientes con sedes.",
		Arguments: arguments("products", "period?", "sites?", "clients?"),
		Examples: []string{"comparar las ventas diarias de dos productos", "ventas por día de unos productos", "evolución diaria de varios productos",
			"cómo se vendieron estos productos cada día", "gráfico diario por producto", "venta diaria de cada producto",
			"línea de ventas diarias de varios productos", "comparar productos día a día"},
		Message: "Ventas diarias por producto {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("date", "product_id", "amount", "line")
		},
	},
	{
		Name: "sales_by_product_and_week", Domain: "sales", Access: salesAccess,
		Title: "Ventas semanales por producto", Gist: "una línea por producto con su monto vendido cada semana",
		Description: "Una línea por cada producto pedido, hasta 12, con el monto vendido cada semana (de lunes a domingo), en soles y sin ventas anuladas. " +
			"No combina clientes con sedes.",
		Arguments: arguments("products", "period?", "sites?", "clients?"),
		Examples: []string{"comparar las ventas semanales de productos", "ventas por semana de unos productos", "evolución semanal de varios productos",
			"cómo se vendió cada producto por semana", "gráfico semanal por producto", "venta semanal de cada producto",
			"tendencia semanal de dos productos", "comparar productos semana a semana"},
		Message: "Ventas semanales por producto {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("WEEK(date)", "product_id", "amount", "line")
		},
	},
	{
		Name: "sales_by_product_and_month", Domain: "sales", Access: salesAccess,
		Title: "Ventas mensuales por producto", Gist: "barras por producto con su monto vendido cada mes",
		Description: "Barras por cada producto pedido, hasta 12, con el monto vendido cada mes, en soles y sin ventas anuladas. No combina clientes con sedes.",
		Arguments:   arguments("products", "period?", "sites?", "clients?"),
		Examples: []string{"comparar las ventas mensuales de productos", "ventas por mes de unos productos", "evolución mensual de varios productos",
			"cómo se vendió cada producto por mes", "gráfico mensual por producto", "venta mensual de cada producto",
			"tendencia mensual de dos productos", "comparar productos mes a mes"},
		Message: "Ventas mensuales por producto {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("MONTH(date)", "product_id", "amount", "bar")
		},
	},
	{
		Name: "sales_by_client_and_month", Domain: "sales", Access: salesAccess,
		Title: "Compras mensuales por cliente", Gist: "barras por cliente con su monto comprado cada mes",
		Description: "Barras por cada cliente pedido, hasta 12, con el monto que compró cada mes, en soles y sin ventas anuladas. " +
			"No hay datos por cliente y sede a la vez, ni número de compras.",
		Arguments: arguments("clients", "period?", "products?"),
		Examples: []string{"compras mensuales de unos clientes", "cuánto me compró cada mes un cliente", "evolución mensual de un cliente",
			"comparar clientes mes a mes", "compras por mes de varios clientes", "cómo viene comprando un cliente cada mes",
			"historial mensual de compras de clientes", "gráfico mensual por cliente",
			"compras de cada cliente mes a mes", "cuánto compra un cliente por mes",
			"ventas mensuales a un cliente"},
		Message: "Compras mensuales por cliente {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("MONTH(date)", "client_id", "amount", "bar")
		},
	},
	{
		Name: "sales_by_category_and_week", Domain: "sales", Access: salesAccess,
		Title: "Ventas semanales por categoría", Gist: "una línea por categoría con su monto vendido cada semana",
		Description: "Una línea por cada categoría pedida, hasta 12, con el monto vendido cada semana, en soles y sin ventas anuladas. " +
			"Un producto en dos categorías cuenta en ambas, así que las categorías suman más que el total de ventas.",
		Arguments: arguments("categories", "period?", "sites?"),
		Examples: []string{"ventas semanales por categoría", "evolución semanal de unas categorías", "comparar categorías semana a semana",
			"cómo se vendió cada categoría por semana", "tendencia semanal de una categoría", "gráfico semanal de categorías",
			"ventas por semana de una línea de productos", "categorías semana a semana"},
		Message: "Ventas semanales por categoría {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.timelinePerRecord("WEEK(date)", "category_id", "amount", "line")
			shown.note = categoryOverlapNote
			return shown, err
		},
	},
	{
		Name: "sales_by_site_and_week", Domain: "sales", Access: salesAccess,
		Title: "Ventas semanales por sede", Gist: "una línea por sede con su monto vendido cada semana",
		Description: "Una línea por sede, hasta 12, con el monto vendido cada semana, en soles y sin ventas anuladas; las ventas sin almacén salen como «(ninguno)». " +
			"No hay datos por sede y cliente a la vez.",
		Arguments: arguments("period?", "sites?", "products?"),
		Examples: []string{"ventas semanales por sede", "evolución semanal de cada local", "comparar sedes semana a semana",
			"cómo vendió cada tienda por semana", "tendencia semanal de las sucursales", "gráfico semanal por sede",
			"ventas por semana de cada local", "sedes semana a semana"},
		Message: "Ventas semanales por sede {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("WEEK(date)", "site_id", "amount", "line")
		},
	},

	// ─── Comparisons: two statements, joined in Go ───────────────────────────────
	{
		Name: "sales_vs_previous_period", Domain: "sales", Access: salesAccess,
		Title: "Ventas frente al periodo anterior", Gist: "total vendido comparado con otro periodo, con diferencia y variación",
		Description: "Crecimiento o caída de las ventas: compara el monto vendido en el periodo con el de otro periodo, por defecto el de igual duración justo antes: ambos totales, " +
			"la diferencia y la variación en porcentaje; en soles y sin ventas anuladas. No combina clientes con sedes.",
		Arguments: arguments("period?", "compare_period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"comparar las ventas con el periodo anterior", "vendí más o menos que antes", "crecieron las ventas", "variación de las ventas",
			"cuánto subieron o bajaron las ventas", "ventas frente al mes anterior", "comparación de ventas entre dos periodos", "cayeron las ventas",
			"crecimiento de las ventas frente al periodo anterior", "variación porcentual de las ventas", "cuánto crecimos en ventas"},
		Message: "Ventas {periodo} comparadas con las {comparado}{filtros}: {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.compareTotals() },
	},
	{
		Name: "sales_by_product_vs_previous_period", Domain: "sales", Access: salesAccess,
		Title: "Ventas por producto frente al periodo anterior", Gist: "monto de cada producto en ambos periodos y su variación",
		Description: "Crecimiento o caída de cada producto: el monto vendido en el periodo y en otro periodo, por defecto el de igual duración justo antes, con la variación en porcentaje; " +
			"en soles y sin ventas anuladas. No combina clientes con sedes.",
		Arguments: arguments("period?", "compare_period?", "clients?", "categories?", "sites?", "top?"),
		Examples: []string{"qué productos crecieron en ventas", "productos que bajaron sus ventas", "comparar las ventas por producto con el periodo anterior",
			"variación de ventas por producto", "productos que más subieron", "cómo cambió la venta de cada producto", "productos con caída de ventas",
			"comparativo de productos entre dos periodos",
			"variación porcentual de cada producto frente al periodo anterior", "crecimiento de las ventas de cada producto", "productos cuya venta cayó o subió",
			"comparativo de ventas por producto frente a otro periodo"},
		Message: "Ventas por producto {periodo} comparadas con las {comparado}{filtros}.",
		run:     func(call *functionCall) (shownResult, error) { return call.compareByRecord("product_id", call.top) },
	},
	{
		Name: "sales_by_client_vs_previous_period", Domain: "sales", Access: salesAccess,
		Title: "Compras por cliente frente al periodo anterior", Gist: "compras de cada cliente en ambos periodos y su variación",
		Description: "Crecimiento o caída de cada cliente: el monto que compró en el periodo y en otro periodo, por defecto el de igual duración justo antes, con la variación en porcentaje; " +
			"en soles y sin ventas anuladas. No hay datos por cliente y sede a la vez.",
		Arguments: arguments("period?", "compare_period?", "products?", "top?"),
		Examples: []string{"qué clientes compraron menos que antes", "clientes que aumentaron sus compras", "comparar las compras por cliente con el periodo anterior",
			"variación de compras por cliente", "clientes que bajaron sus compras", "cómo cambiaron las compras de cada cliente",
			"comparativo de clientes entre dos periodos", "clientes que crecieron",
			"variación porcentual de las compras de cada cliente", "crecimiento de las compras de cada cliente", "clientes cuyas compras cayeron",
			"variación de lo que compró cada cliente", "comparar lo vendido a cada cliente con otro periodo"},
		Message: "Compras por cliente {periodo} comparadas con las {comparado}{filtros}.",
		run:     func(call *functionCall) (shownResult, error) { return call.compareByRecord("client_id", call.top) },
	},
	{
		Name: "sales_by_category_vs_previous_period", Domain: "sales", Access: salesAccess,
		Title: "Ventas por categoría frente al periodo anterior", Gist: "monto de cada categoría en ambos periodos y su variación",
		Description: "Crecimiento o caída de cada categoría de producto: el monto vendido en el periodo y en otro periodo, por defecto el de igual duración justo antes, con la variación; " +
			"en soles y sin ventas anuladas. Un producto en dos categorías cuenta en ambas, así que las categorías suman más que el total.",
		Arguments: arguments("period?", "compare_period?", "sites?", "top?"),
		Examples: []string{"qué categorías crecieron", "categorías que bajaron sus ventas", "comparar las ventas por categoría con el periodo anterior",
			"variación de ventas por categoría", "cómo cambió cada línea de productos", "categorías con caída de ventas",
			"comparativo de categorías entre dos periodos", "familias de productos que más subieron"},
		Message: "Ventas por categoría {periodo} comparadas con las {comparado}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.compareByRecord("category_id", call.top)
			shown.note = categoryOverlapNote
			return shown, err
		},
	},
	{
		Name: "sales_by_site_vs_previous_period", Domain: "sales", Access: salesAccess,
		Title: "Ventas por sede frente al periodo anterior", Gist: "monto de cada sede en ambos periodos y su variación",
		Description: "Crecimiento o caída de cada sede: el monto vendido en el periodo y en otro periodo, por defecto el de igual duración justo antes, con la variación en porcentaje; " +
			"en soles y sin ventas anuladas. No hay datos por sede y cliente a la vez.",
		Arguments: arguments("period?", "compare_period?", "products?"),
		Examples: []string{"qué sede creció más", "sedes que bajaron sus ventas", "comparar las ventas por sede con el periodo anterior",
			"variación de ventas por local", "cómo cambiaron las ventas de cada tienda", "sucursales con caída de ventas",
			"comparativo de sedes entre dos periodos", "locales que más subieron",
			"crecimiento de las ventas de cada sede", "variación porcentual por sede",
			"variación de lo vendido en cada local"},
		Message: "Ventas por sede {periodo} comparadas con las {comparado}{filtros}.",
		run:     func(call *functionCall) (shownResult, error) { return call.compareByRecord("site_id", MaxTableRows) },
	},
	{
		Name: "products_stopped_selling", Domain: "sales", Access: salesAccess,
		Title: "Productos que dejaron de venderse", Gist: "productos con ventas en un periodo anterior y ninguna en el periodo",
		Description: "Productos que tuvieron ventas en el periodo de comparación, por defecto el de igual duración justo antes, y ninguna en el periodo, " +
			"con lo que vendieron antes en soles; sin ventas anuladas.",
		Arguments: arguments("period?", "compare_period?", "sites?"),
		Examples: []string{"productos que dejaron de venderse", "qué productos ya no se venden", "productos sin ventas recientes",
			"productos que no se vendieron este periodo", "artículos que dejaron de salir", "productos que antes se vendían y ahora no",
			"productos parados", "qué ya no sale"},
		Message: "Productos vendidos {comparado} que no se vendieron {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.stopped("product_id", "Ventas")
			shown.emptyMessage = "Todos los productos vendidos {comparado} también se vendieron {periodo}."
			return shown, err
		},
	},
	{
		Name: "clients_stopped_buying", Domain: "sales", Access: salesAccess,
		Title: "Clientes que dejaron de comprar", Gist: "clientes con compras en un periodo anterior y ninguna en el periodo",
		Description: "Clientes que compraron en el periodo de comparación, por defecto el de igual duración justo antes, y nada en el periodo, " +
			"con lo que compraron antes en soles; sin ventas anuladas ni ventas sin cliente.",
		Arguments: arguments("period?", "compare_period?"),
		Examples: []string{"clientes que dejaron de comprar", "qué clientes ya no me compran", "clientes perdidos", "clientes inactivos",
			"clientes que no volvieron a comprar", "quiénes compraban antes y ahora no", "clientes sin compras recientes", "clientes que se fueron"},
		Message: "Clientes que compraron {comparado} y no compraron {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.stopped("client_id", "Compras")
			shown.emptyMessage = "Todos los clientes que compraron {comparado} también compraron {periodo}."
			return shown, err
		},
	},

	// ─── Unpaid amount of the period's sales (debt_amount) ───────────────────────
	{
		Name: "unpaid_sales_total", Domain: "sales", Access: salesAccess,
		Title: "Por cobrar de las ventas del periodo", Gist: "monto aún no pagado de las ventas hechas en el periodo",
		Description: "Lo que falta cobrar de las ventas hechas en el periodo, en soles y sin ventas anuladas. " +
			"No es la deuda total del cliente: las ventas de otros periodos no cuentan; no combina clientes con sedes.",
		Arguments: arguments("period?", "products?", "clients?", "categories?", "sites?"),
		Examples: []string{"cuánto me deben de las ventas", "ventas pendientes de cobro", "monto por cobrar de las ventas", "cuánto falta cobrar",
			"ventas al crédito sin pagar", "cuánto no me han pagado", "saldo pendiente de las ventas", "cuánto me debe un cliente por sus compras",
			"total pendiente de cobro", "monto adeudado de las ventas",
			"total de ventas sin cobrar"},
		Message: "Por cobrar de las ventas {periodo}{filtros}: {valor}.",
		run:     func(call *functionCall) (shownResult, error) { return call.total("debt_amount") },
	},
	{
		Name: "unpaid_sales_by_client", Domain: "sales", Access: salesAccess,
		Title: "Por cobrar por cliente", Gist: "clientes con monto aún no pagado de las ventas del periodo",
		Description: "Ranking de clientes por lo que falta cobrar de sus compras del periodo, en soles y sin ventas anuladas. " +
			"No es la deuda total del cliente: las ventas de otros periodos no cuentan.",
		Arguments: arguments("period?", "products?", "top?"),
		Examples: []string{"quién me debe", "clientes que deben", "clientes con ventas por cobrar", "ranking de deudores", "qué clientes no han pagado",
			"cuánto debe cada cliente por las ventas", "clientes con saldo pendiente", "deudas de clientes por ventas",
			"clientes deudores", "cuánto me debe cada cliente",
			"clientes con deuda"},
		Message: "Clientes con ventas por cobrar {periodo}{filtros}: {valor} en total.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("client_id", []string{"debt_amount"}, "debt_amount", call.top)
		},
	},
	{
		Name: "unpaid_sales_by_week", Domain: "sales", Access: salesAccess,
		Title: "Por cobrar por semana de venta", Gist: "barras del monto no pagado según la semana de la venta",
		Description: "Barras con lo que falta cobrar de las ventas de cada semana del periodo, en soles y sin ventas anuladas. " +
			"No es la deuda total de los clientes: las ventas de otros periodos no cuentan.",
		Arguments: arguments("period?", "clients?"),
		Examples: []string{"por cobrar por semana", "de qué semanas son las ventas sin pagar", "ventas no pagadas por semana", "deuda de ventas semana a semana",
			"evolución semanal de lo pendiente de cobro", "cuánto falta cobrar de cada semana", "gráfico semanal de ventas por cobrar",
			"saldo pendiente de cada semana de ventas",
			"ventas sin cobrar de cada semana", "por cobrar según la semana de venta", "semanas con ventas impagas"},
		Message: "Por cobrar de las ventas de cada semana {periodo}{filtros}: {valor} en total.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timeline("WEEK(date)", "debt_amount", "bar")
		},
	},
	{
		Name: "unpaid_sales_by_site", Domain: "sales", Access: salesAccess,
		Title: "Por cobrar por sede", Gist: "monto no pagado de las ventas del periodo en cada sede",
		Description: "Barras con lo que falta cobrar de las ventas del periodo de cada sede, en soles y sin ventas anuladas. " +
			"No es la deuda total de los clientes: las ventas de otros periodos no cuentan.",
		Arguments: arguments("period?"),
		Examples: []string{"por cobrar por sede", "qué sede tiene más ventas sin pagar", "ventas pendientes de cobro por local", "cuánto falta cobrar en cada tienda",
			"deuda de ventas por sucursal", "saldo por cobrar de cada sede", "cobranza pendiente por local", "sedes con más ventas al crédito sin pagar"},
		Message: "Por cobrar de las ventas por sede {periodo}{filtros}: {valor} en total.",
		run: func(call *functionCall) (shownResult, error) {
			shown, err := call.ranking("site_id", []string{"debt_amount"}, "debt_amount", MaxTableRows)
			return asBars(shown, "site_id", "debt_amount"), err
		},
	},

	// ─── Undelivered quantity (pending_quantity), always per product: rule Q ─────
	{
		Name: "undelivered_by_product", Domain: "sales", Access: salesAccess,
		Title: "Unidades por entregar por producto", Gist: "productos con unidades vendidas aún no entregadas",
		Description: "Ranking de productos por las unidades vendidas en el periodo que aún no se entregan, cada producto en su propia unidad; sin ventas anuladas. " +
			"No combina clientes con sedes.",
		Arguments: arguments("period?", "clients?", "sites?", "top?"),
		Examples: []string{"productos pendientes de entrega", "qué falta entregar", "unidades por despachar", "pedidos sin entregar por producto",
			"qué productos se deben entregar", "mercadería vendida no entregada", "pendientes de despacho", "cuánto falta entregar de cada producto",
			"productos con entregas pendientes", "unidades pendientes de entrega de cada producto", "lo que falta despachar por producto",
			"productos que aún no se entregan", "cuánto queda por entregar de cada producto"},
		Message: "Unidades por entregar de las ventas {periodo}{filtros}, por producto.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("product_id", []string{"pending_quantity"}, "pending_quantity", call.top)
		},
	},
	{
		Name: "undelivered_by_client_and_product", Domain: "sales", Access: salesAccess,
		Title: "Por entregar por cliente y producto", Gist: "tabla de clientes y productos con unidades aún no entregadas",
		Description: "Tabla de cliente y producto con las unidades vendidas en el periodo que aún no se entregan, cada producto en su propia unidad; " +
			"sin ventas anuladas. No hay datos por cliente y sede a la vez.",
		Arguments: arguments("period?", "clients?", "products?"),
		Examples: []string{"qué le falta entregar a cada cliente", "entregas pendientes por cliente", "clientes con productos por entregar",
			"qué le debo entregar a un cliente", "despachos pendientes por cliente", "pendientes de entrega de cada cliente",
			"productos por entregar a clientes", "lista de entregas pendientes por cliente",
			"unidades pendientes de entrega de un cliente",
			"productos pendientes de entregar a un cliente"},
		Message: "Unidades por entregar de las ventas {periodo}{filtros}, por cliente y producto.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("client_id, product_id", []string{"pending_quantity"}, "pending_quantity", MaxTableRows)
		},
	},
	{
		Name: "undelivered_by_site_and_product", Domain: "sales", Access: salesAccess,
		Title: "Por entregar por sede y producto", Gist: "tabla de sedes y productos con unidades aún no entregadas",
		Description: "Tabla de sede y producto con las unidades vendidas en el periodo que aún no se entregan, cada producto en su propia unidad; " +
			"sin ventas anuladas. No hay datos por sede y cliente a la vez.",
		Arguments: arguments("period?", "sites?", "products?"),
		Examples: []string{"qué falta entregar en cada sede", "entregas pendientes por local", "productos por despachar en cada tienda",
			"pendientes de entrega por sucursal", "despachos pendientes de una sede", "unidades por entregar por local y producto",
			"qué debe entregar cada sede", "lista de pendientes por sede",
			"unidades por despachar de cada almacén",
			"almacenes con productos por entregar"},
		Message: "Unidades por entregar de las ventas {periodo}{filtros}, por sede y producto.",
		run: func(call *functionCall) (shownResult, error) {
			return call.ranking("site_id, product_id", []string{"pending_quantity"}, "pending_quantity", MaxTableRows)
		},
	},
	{
		Name: "undelivered_by_product_and_week", Domain: "sales", Access: salesAccess,
		Title: "Por entregar por semana y producto", Gist: "una línea por producto con unidades sin entregar por semana de venta",
		Description: "Una línea por cada producto pedido, hasta 12, con las unidades aún no entregadas de las ventas de cada semana, en la unidad de cada producto; " +
			"sin ventas anuladas.",
		Arguments: arguments("products", "period?", "sites?"),
		Examples: []string{"pendientes de entrega por semana de un producto", "evolución semanal de lo no entregado", "unidades por entregar cada semana",
			"de qué semanas son las entregas pendientes", "despachos pendientes semana a semana", "gráfico semanal de pendientes de entrega",
			"cuánto falta entregar por semana de venta", "entregas atrasadas por semana",
			"evolución de las entregas pendientes"},
		Message: "Unidades por entregar por semana de venta {periodo}{filtros}.",
		run: func(call *functionCall) (shownResult, error) {
			return call.timelinePerRecord("WEEK(date)", "product_id", "pending_quantity", "line")
		},
	},
}

// categoryOverlapNote follows any answer per category.
const categoryOverlapNote = "Un producto en varias categorías cuenta en cada una: las categorías suman más que el total de ventas."
