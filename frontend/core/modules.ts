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
    { name: "Configuration|Configuración", minName: "CON", id: 1, icon: "icon-flow-merge",
      options: [
				{
					name: "My Company|Mi Empresa", route: "/configuracion/parametros", icon: "icon-cog",
					descripcion: "Edita los datos de tu empresa, pasarela de pago, envío de correos."
        },
				{
					name: "Users|Usuarios", route: "/seguridad/usuarios", icon: "icon-adult",
					descripcion: "Gestiona usuario y asígnales perfiles."
				},
				{
					name: "Profiles & Access|Perfiles & Accesos", route: "/seguridad/perfiles-accesos",
					descripcion: "Crea perfiles y asígnales accesos.",
          icon: "icon-shield",
				},
				{
					name: "Backups|Backups", route: "/configuracion/backups", icon: "icon-database",
					descripcion: "Descarga y genera respaldos de tu información.",
				},
      ]
    },
    { name: "System|System", minName: "SYS", id: 9, icon: "icon-shield",
      options: [
				{
					name: "Companies|Empresas", route: "/system/empresas", icon: "icon-shield",
					onlySaaS: true,
				},
				{
					name: "Server Panel|Server Panel", route: "/system/server-panel", icon: "icon-shield",
					onlySaaS: true,
				},
				{
					name: "Cron Actions|Acciones Cron", route: "/system/cron-actions", icon: "icon-shield",
					onlySaaS: true,
				},
				{
					name: "Testing|Testing", route: "/system/testing", icon: "icon-shield",
					onlySaaS: true,
				},
      ]
    },
    { name: "Business|Negocio", minName: "NEG",  id: 2, icon: "icon-cube",
      options: [
        { name: "Sites & Warehouses|Sedes & Almacenes", route: "/negocio/sedes-almacenes",
					icon: "icon-home-1",
					descripcion: "Crea sedes y almacenes. Crea los layouts de tus almacenes."
        },
				{
					name: "Products|Productos", route: "/negocio/productos",
					descripcion: "Crea productos y agrúpalos por categoría y marca. Edita precios, unidades y presentaciones de tus productos.",
          icon: "icon-cube"
				},
				{ name: "Customers|Clientes", route: "/negocio/clientes",
          icon: "icon-user"
        },
				{ name: "Suppliers|Proveedores", route: "/negocio/proveedores",
          icon: "icon-truck"
        },
      ]
    },
    { name: "Commercial|Comercial", minName: "Com",  id: 3, icon: "icon-tasks",
      options: [
        { name: "Point of Sale|Punto de Venta", route: "/comercial/sale_order_create",
          icon: "icon-flash"
				},
				{ name: "Sales Management|Gestión Ventas", route: "/comercial/sale_orders_status",
          icon: "icon-flash"
				},
				{ name: "Sales Charts|Gráficos Ventas", route: "/comercial/sale_orders_charts",
          icon: "icon-chart-bar"
				},
				{ name: "Sales Report|Reporte Ventas", route: "/comercial/reporte-ventas",
          icon: "icon-chart-bar"
				},
				{ name: "Shipping Costs|Costos de Envio", route: "/comercial/shipping-costs",
          icon: "icon-flash"
        },
				{ name: "Sales Planning|Proyección Ventas", route: "/comercial/sale_planning",
          icon: "icon-chart-bar"
        },
      ]
		},
		{ name: "Logistics|Logística", minName: "LOG",  id: 4, icon: "icon-tasks",
      options: [
	      { name: "Stock Changes|Cambios Stock", route: "/logistica/products-stock",
	        icon: "icon-chart-bar"
	      },
	      { name: "Purchase Management|Gestión de Compras", route: "/logistica/gestion-compras",
	        icon: "icon-basket"
	      },
	      { name: "Movements Report|Rep. Movimientos", route: "/logistica/almacen-movimientos",
	        icon: "icon-truck"
				},
		    { name: "Purchase Orders|Órdenes Compra", route: "/logistica/purchase-orders",
		      icon: "icon-truck"
		    },
		    { name: "Supplies|Suministros", route: "/logistica/supplies-materials",
		      icon: "icon-cube"
		    },
      ]
    },
		{ name: "Finances|Finanzas", minName: "FIN",  id: 5, icon: "icon-tasks",
      options: [
	      { name: "Cash & Banks|Cajas & Bancos", route: "/finance/cajas",
	        icon: "icon-suitcase"
	      },
	      { name: "Cash Movements|Cajas Movimientos", route: "/finance/cajas-movimientos",
	        icon: "icon-exchange"
				},
		    { name: "Expenses|Gastos", route: "/finance/expenses",
		      icon: "icon-exchange"
				},
		    { name: "Accounts Management|Gestión de Cuentas", route: "/finance/gestion-cuentas",
		      icon: "icon-exchange"
		    },
		    { name: "Cash Flow|Flujo de Caja", route: "/finance/flujo-de-caja",
		      icon: "icon-exchange"
				},
      ]
    },
    { name: "Store|Tienda", minName: "TIE",  id: 7, icon: "icon-buffer",
      options: [
        { name: "Home|Inicio", route: "/tienda/builder-store"
        },
        { name: "About Us|Nosotros", route: "/store-builder/page-2"
        },
        { name: "Store|Tienda", route: "/store-builder/page-3"
        },
        { name: "Product|Producto", route: "/store-builder/page-4"
        },
        { name: "Shopping Cart|Carrito Compra", route: "/store-builder/page-5"
        },
      ]
    },
    { name: "Accounting|Contabilidad", minName: "CNT",  id: 8, icon: "icon-tasks",
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
