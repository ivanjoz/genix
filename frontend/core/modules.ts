import type { IMenuRecord } from '$core/types/modules';

export interface IModule {
  name: string
  id: number
  code: string
  menus: IMenuRecord[]
}

export const AdminModule: IModule = {
  name: "Administration|Administración",
  id: 1,
  code: "admin",
  menus: [
    { name: "My Company|Mi Empresa", minName: "EMP", id: 1, icon: "icon-[fa--sitemap]",
      options: [
				{
					name: "Configuration|Configuración", route: "/company/configuration", icon: "icon-[fa--cog]",
					descripcion: "Edita los datos de tu empresa, pasarela de pago, envío de correos. Genera y descarga backups."
        },
        { name: "Sites & Warehouses|Sedes & Almacenes", route: "/business/branches-warehouses",
					icon: "icon-[fa--home]",
					descripcion: "Crea sedes y almacenes. Crea los layouts de tus almacenes."
        },
				{
					name: "Users & Profiles|Usuarios & Perfiles", route: "/security/users-profiles",
					descripcion: "Gestiona usuarios, crea perfiles y asígnales accesos.",
          icon: "icon-[fa--user-secret]",
				},
      ]
    },
    { name: "System|System", minName: "SYS", id: 9, icon: "icon-[fa--cogs]",
      options: [
				{
					name: "Companies|Empresas", route: "/system/companies", icon: "icon-[fa--building]",
					onlySaaS: true,
				},
				{
					name: "Server Panel|Server Panel", route: "/system/server-panel", icon: "icon-[fa--server]",
					onlySaaS: true,
				},
				{
					name: "Observability|Observabilidad", route: "/system/observability", icon: "icon-[fa--bar-chart]",
					onlySaaS: true,
				},
				{
					name: "Cron Actions|Acciones Cron", route: "/system/cron-actions", icon: "icon-[fa--clock-o]",
					onlySaaS: true,
				},
				{
					name: "Developer|Developer", route: "/system/developer", icon: "icon-[fa--flask]",
					onlySaaS: true,
				},
      ]
    },
    { name: "Production|Producción", minName: "PRD",  id: 10, icon: "icon-[fa--cubes]",
      options: [
				{
					name: "Products|Productos", route: "/production/products",
					descripcion: "Crea productos y agrúpalos por categoría y marca. Edita precios, unidades y presentaciones de tus productos.",
          icon: "icon-[fa--cube]"
				},
				{ name: "Supplies & Materials|Insumos & Materiales", route: "/production/supplies-materials",
					descripcion: "Registra los insumos y materiales que consume tu producción, con sus proveedores y stock mínimo.",
					icon: "icon-[fa--flask]"
				},
      ]
    },
    { name: "Commercial|Comercial", minName: "Com",  id: 3, icon: "icon-[fa--tasks]",
      options: [
        { name: "Point of Sale|Punto de Venta", route: "/sales/sale_order_create",
          icon: "icon-[fa--bolt]"
				},
				{ name: "Sales Management|Gestión Ventas", route: "/sales/sale_orders_status",
          icon: "icon-[fa--bolt]"
				},
				{ name: "Sales Charts|Gráficos Ventas", route: "/sales/sale_orders_charts",
          icon: "icon-[fa--bar-chart]"
				},
				{ name: "Sales Report|Reporte Ventas", route: "/sales/sales-report",
          icon: "icon-[fa--bar-chart]"
				},
				{ name: "Shipping Costs|Costos de Envio", route: "/sales/shipping-costs",
          icon: "icon-[fa--bolt]"
        },
				{ name: "Sales Planning|Proyección Ventas", route: "/sales/sale_planning",
          icon: "icon-[fa--bar-chart]"
        },
      ]
		},
    { name: "Clients (CRM)|Clientes (CRM)", minName: "CRM",  id: 11, icon: "icon-[fa--users]",
      options: [
				{ name: "Customers|Clientes", route: "/crm/customers",
					descripcion: "Registra tus clientes con RUC o DNI y consulta su historial de compras.",
          icon: "icon-[fa--user]"
        },
      ]
    },
		{ name: "Logistics|Logística", minName: "LOG",  id: 4, icon: "icon-[fa--tasks]",
      options: [
	      { name: "Stock Changes|Cambios Stock", route: "/logistics/products-stock",
	        icon: "icon-[fa--bar-chart]"
	      },
	      { name: "Purchase Management|Gestión de Compras", route: "/logistics/purchase-management",
	        icon: "icon--supermarket-cart"
	      },
	      { name: "Movements Report|Rep. Movimientos", route: "/logistics/warehouse-movements",
	        icon: "icon-[fa--truck]"
				},
		    { name: "Purchase Orders|Órdenes Compra", route: "/logistics/purchase-orders",
		      icon: "icon-[fa--truck]"
		    },
		    { name: "Suppliers|Proveedores", route: "/logistics/suppliers",
		      icon: "icon-[fa--truck]"
		    },
      ]
    },
		{ name: "Finances|Finanzas", minName: "FIN",  id: 5, icon: "icon-[fa--tasks]",
      options: [
	      { name: "Cash & Banks|Cajas & Bancos", route: "/finance/cash-banks",
	        icon: "icon-[fa--briefcase]"
	      },
	      { name: "Cash Movements|Cajas Movimientos", route: "/finance/cash-banks-movements",
	        icon: "icon-[fa--exchange]"
				},
		    { name: "Expenses|Gastos", route: "/finance/expenses",
		      icon: "icon-[fa--exchange]"
				},
		    { name: "Accounts Management|Gestión de Cuentas", route: "/finance/gestion-cuentas",
		      icon: "icon-[fa--exchange]"
		    },
		    { name: "Cash Flow|Flujo de Caja", route: "/finance/flujo-de-caja",
		      icon: "icon-[fa--exchange]"
				},
      ]
    },
    { name: "Website|Tienda /Web", minName: "WEB",  id: 7, icon: "icon-[fa--th-large]",
      options: [
        { name: "Pages|Páginas", route: "/webpage-builder/pages"
        },
        { name: "Gallery|Galería", route: "/webpage-builder/gallery"
        },
      ]
    },
    { name: "Accounting|Contabilidad", minName: "CNT",  id: 8, icon: "icon-[fa--tasks]",
      options: [
        { name: "Invoicing|Facturación", route: "/accounting/invoicing"
        },
        { name: "Financial Statements|Estados Financieros",
				},
				{ name: "Balance|Balance",
				},
				{ name: "Assets|Activos", route: "/accounting/assets"
        },
      ]
    },
  ]
}

// El módulo SYSTEM administra la plataforma entera (empresas, servidor, crons), no un tenant:
// sólo la company dueña del SaaS lo ve. El backend repite la restricción por endpoint.
export const SAAS_COMPANY_ID = 1

const saasOnlyRoutes = new Set(
  AdminModule.menus.flatMap(menu =>
    (menu.options || []).filter(option => option.onlySaaS && option.route).map(option => option.route!)
  )
)

export const isSaaSOnlyRoute = (route: string) => saasOnlyRoutes.has(route)

const Modules: IModule[] = [AdminModule]
export default Modules
