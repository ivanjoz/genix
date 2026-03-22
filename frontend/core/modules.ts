import type { IMenuRecord } from '$core/types/modules';

export interface IModule {
  name: string
  id: number
  code: string
  menus: IMenuRecord[]
}

export const AdminModule: IModule = {
  name: "Administración",
  id: 1,
  code: "admin",
  menus: [
    { name: "CONFIGURACIÓN", minName: "CON", id: 1, icon: "icon-flow-merge",
      options: [
				{
					name: "Mi Empresa", route: "/configuracion/parametros", icon: "icon-cog",
					descripcion: "Edita los datos de tu empresa, pasarela de pago, envío de correos."
        },
				{
					name: "Usuarios", route: "/seguridad/usuarios", icon: "icon-adult",
					descripcion: "Gestiona usuario y asígnales perfiles."
				},
				{
					name: "Perfiles & Accesos", route: "/seguridad/perfiles-accesos",
					descripcion: "Crea perfiles y asígnales accesos.",
          icon: "icon-shield",
				},
				{
					name: "Backups", route: "/configuracion/backups", icon: "icon-database",
					descripcion: "Descarga y genera respaldos de tu información.",
				},
				{
					name: "Empresas", route: "/configuracion/empresas", icon: "icon-shield",
					onlySaaS: true,
				},
				{
					name: "Server Panel", route: "/configuracion/server-panel", icon: "icon-shield",
					onlySaaS: true,
				},
      ]
    },
    { name: "NEGOCIO", minName: "NEG",  id: 2, icon: "icon-cube",
      options: [
        { name: "Sedes & Almacenes", route: "/negocio/sedes-almacenes",
					icon: "icon-home-1",
					descripcion: "Crea sedes y almacenes. Crea los layouts de tus almacenes."
        },
				{
					name: "Productos", route: "/negocio/productos",
					descripcion: "Crea productos y agrúpalos por categoría y marca. Edita precios, unidades y presentaciones de tus productos.",
          icon: "icon-cube"
				},
				{ name: "Clientes", route: "/negocio/clientes",
          icon: "icon-user"
        },
				{ name: "Proveedores", route: "/negocio/proveedores",
          icon: "icon-truck"
        },
      ]
    },
    { name: "Comercial", minName: "Com",  id: 3, icon: "icon-tasks",
      options: [
        { name: "Punto de Venta", route: "/comercial/sale_order_create",
          icon: "icon-flash"
				},
				{ name: "Gestión Ventas", route: "/comercial/sale_orders_status",
          icon: "icon-flash"
				},
				{ name: "Gráficos Ventas", route: "/comercial/sale_orders_charts",
          icon: "icon-chart-bar"
				},
				{ name: "Costos de Envio", route: "/comercial/shipping-costs",
          icon: "icon-flash"
        },
      ]
		},
		{ name: "Logística", minName: "LOG",  id: 4, icon: "icon-tasks",
      options: [
	      { name: "Gestión de Stock", route: "/logistica/products-stock",
	        icon: "icon-chart-bar"
	      },
	      { name: "Rep. Movimientos", route: "/logistica/almacen-movimientos",
	        icon: "icon-truck"
				},
		    { name: "Órdenes Compra", route: "/logistica/ordenes-de-compra",
		      icon: "icon-truck"
		    },
      ]
    },
		{ name: "Finanzas", minName: "FIN",  id: 5, icon: "icon-tasks",
      options: [
	      { name: "Cajas & Bancos", route: "/finanzas/cajas",
	        icon: "icon-suitcase"
	      },
	      { name: "Cajas Movimientos", route: "/finanzas/cajas-movimientos",
	        icon: "icon-exchange"
				},
		    { name: "Gastos", route: "/finanzas/gastos",
		      icon: "icon-exchange"
				},
		    { name: "Gestión de Cuentas", route: "/finanzas/gestion-cuentas",
		      icon: "icon-exchange"
		    },
		    { name: "Flujo de Caja", route: "/finanzas/flujo-de-caja",
		      icon: "icon-exchange"
				},
      ]
    },
    { name: "TIENDA", minName: "TIE",  id: 7, icon: "icon-buffer",
      options: [
        { name: "Inicio", route: "/tienda/builder-store"
        },
        { name: "Nosotros", route: "/store-builder/page-2"
        },
        { name: "Tienda", route: "/store-builder/page-3"
        },
        { name: "Producto", route: "/store-builder/page-4"
        },
        { name: "Carrito Compra", route: "/store-builder/page-5"
        },
      ]
    },
    { name: "Contabilidad", minName: "CNT",  id: 8, icon: "icon-tasks",
      options: [
        { name: "Facturación",
        },
        { name: "Estados Financieros",
				},
				{ name: "Balance",
				},
				{ name: "Activos",
        },
      ]
    },
  ]
}

const Modules: IModule[] = [AdminModule]
export default Modules
