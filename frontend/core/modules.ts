import type { IMenuRecord } from '$core/types/modules';

export interface IModule {
  name: string
  id: number
  code: string
  menus: IMenuRecord[]
  indexedDBTables: {[e: string]: string }
}

export const AdminModule: IModule = {
  name: "Administración",
  id: 1,
  code: "admin",
  indexedDBTables: {
    empresas: "id",
    empresa: "id",
    usuarios: "id",
    seguridad_accesos: "id",
    perfiles: "id",
    fetchedIDBTables: 'key',
    cacheUpdates: 'key,route',
    cacheStatic: 'key',
    backups: 'Name',
    cajas: "ID",
    // Scylla
    productos: "ID",
    sedes_almacenes: '[_pk+ID]',
    productos_stock: '[AlmacenID+ID]',
    listas_compartidas: '[pk+ID]',
    pais_ciudades: '[PaisID+ID]',
    galeria_images: 'Image'
  },
  menus: [
    { name: "CONFIGURACIÓN", minName: "CON", id: 1, icon: "icon-flow-merge",
      options: [
				{
					name: "Mi Empresa", route: "/admin/empresas", icon: "icon-black-tie",
					
        },
        { name: "Parámetros", route: "/admin/parametros", icon: "icon-cog",
        },
        { name: "Usuarios", route: "/admin/usuarios", icon: "icon-adult",
				},
        { name: "Perfiles & Accesos", route: "/admin/perfiles-accesos",
          icon: "icon-shield",
				},
				{
					name: "Server Panel", route: "/admin/server-panel", icon: "icon-shield",
					onlySaaS: true,
				},
        { name: "Backups", route: "/admin/backups", icon: "icon-database"
        },
      ]
    },
    { name: "NEGOCIO", minName: "NEG",  id: 2, icon: "icon-cube",
      options: [
        { name: "Sedes & Almacenes", route: "/operaciones/sedes-almacenes",
          icon: "icon-home-1"
        },
        { name: "Productos", route: "/operaciones/productos",
          icon: "icon-cube"
				},
				{ name: "Clientes & Proveedores", route: "/operaciones/clientes-proveedores",
          icon: "icon-cube"
        },
      ]
    },
    { name: "Comercial", minName: "Com",  id: 3, icon: "icon-tasks",
      options: [
        { name: "Punto de Venta", route: "/operaciones/ventas",
          icon: "icon-flash"
				},
				{ name: "Gestión Ventas", route: "/operaciones/gestion-ventas",
          icon: "icon-flash"
				},
				{ name: "Costos de Envio", route: "/operaciones/shipping-costs",
          icon: "icon-flash"
        },
      ]
		},
		{ name: "Logística", minName: "LOG",  id: 4, icon: "icon-tasks",
      options: [
	      { name: "Gestión de Stock", route: "/operaciones/productos-stock",
	        icon: "icon-chart-bar"
	      },
	      { name: "Rep. Movimientos", route: "/operaciones/almacen-movimientos",
	        icon: "icon-truck"
				},
		    { name: "Órdenes Compra", route: "/logistica/ordenes-de-compra",
		      icon: "icon-truck"
		    },
      ]
    },
		{ name: "Finanzas", minName: "FIN",  id: 5, icon: "icon-tasks",
      options: [
	      { name: "Cajas & Bancos", route: "/operaciones/cajas",
	        icon: "icon-suitcase"
	      },
	      { name: "Cajas Movimientos", route: "/operaciones/cajas-movimientos",
	        icon: "icon-exchange"
				},
		    { name: "Gastos", route: "/operaciones/gastos",
		      icon: "icon-exchange"
				},
		    { name: "Gestión de Cuentas", route: "/operaciones/gestion-cuentas",
		      icon: "icon-exchange"
		    },
		    { name: "Flujo de Caja", route: "/operaciones/flujo-de-caja",
		      icon: "icon-exchange"
				},
      ]
    },
    { name: "TIENDA", minName: "TIE",  id: 7, icon: "icon-buffer",
      options: [
        { name: "Inicio", route: "/builder-store"
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

export const ComercialModule: IModule = {
  name: "Comercial",
  id: 2,
  code: "comercial",
  indexedDBTables: {
    ecommerce_cache: "key"
  },
  menus: [
    { name: "Productos", minName: "PRD",  id: 2, icon: "icon-flow-merge",
      options: [
        { name: "Módulo de Venta",
        },
        { name: "Productos",
        },
        { name: "Almacenes",
        },
      ]
    },
    { name: "Página Web", minName: "PGW",  id: 3, icon: "icon-buffer",
      options: [
        { name: "Inicio", route: "/cms/webpage/1"
        },
        { name: "Nosotros", route: "/cms/webpage/2"
        },
        { name: "Tienda", route: "/cms/webpage/3"
        },
        { name: "Producto", route: "/cms/webpage/4"
        },
        { name: "Carrito Compra", route: "/cms/webpage/5"
        },
      ]
    },
  ]
}

const Modules: IModule[] = [AdminModule, ComercialModule]
export default Modules
