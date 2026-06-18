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
    { name: "Configuration|Configuración", minName: "CON", id: 1, icon: "icon-[fa--sitemap]",
      options: [
				{
					name: "My Company|Mi Empresa", route: "/configuration/parameters", icon: "icon-[fa--cog]",
					descripcion: "Edita los datos de tu empresa, pasarela de pago, envío de correos."
        },
				{
					name: "Users|Usuarios", route: "/security/users", icon: "icon-[fa--user-secret]",
					descripcion: "Gestiona usuario y asígnales perfiles."
				},
				{
					name: "Profiles & Access|Perfiles & Accesos", route: "/security/access-profiles",
					descripcion: "Crea perfiles y asígnales accesos.",
          icon: "icon-[fa--shield]",
				},
				{
					name: "Backups|Backups", route: "/configuration/backups", icon: "icon-[fa--database]",
					descripcion: "Descarga y genera respaldos de tu información.",
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
					name: "Cron Actions|Acciones Cron", route: "/system/cron-actions", icon: "icon-[fa--clock-o]",
					onlySaaS: true,
				},
				{
					name: "Testing|Testing", route: "/system/testing", icon: "icon-[fa--flask]",
					onlySaaS: true,
				},
      ]
    },
    { name: "Business|Negocio", minName: "NEG",  id: 2, icon: "icon-[fa--cube]",
      options: [
        { name: "Sites & Warehouses|Sedes & Almacenes", route: "/business/branches-warehouses",
					icon: "icon-[fa--home]",
					descripcion: "Crea sedes y almacenes. Crea los layouts de tus almacenes."
        },
				{
					name: "Products|Productos", route: "/business/products",
					descripcion: "Crea productos y agrúpalos por categoría y marca. Edita precios, unidades y presentaciones de tus productos.",
          icon: "icon-[fa--cube]"
				},
				{ name: "Customers|Clientes", route: "/business/customers",
          icon: "icon-[fa--user]"
        },
				{ name: "Suppliers|Proveedores", route: "/business/suppliers",
          icon: "icon-[fa--truck]"
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
		    { name: "Supplies|Suministros", route: "/logistics/supplies-materials",
		      icon: "icon-[fa--cube]"
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
    { name: "Website|Página Web", minName: "WEB",  id: 7, icon: "icon-[fa--th-large]",
      options: [
        { name: "Pages|Páginas", route: "/webpage-builder/pages"
        },
        { name: "Gallery|Galería", route: "/webpage-builder/gallery"
        },
      ]
    },
    { name: "Accounting|Contabilidad", minName: "CNT",  id: 8, icon: "icon-[fa--tasks]",
      options: [
        { name: "Invoicing|Facturación",
        },
        { name: "Financial Statements|Estados Financieros",
				},
				{ name: "Balance|Balance",
				},
				{ name: "Assets|Activos",
        },
      ]
    },
  ]
}

const Modules: IModule[] = [AdminModule]
export default Modules
