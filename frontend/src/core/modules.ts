import { IMenuRecord } from "./menu";

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
  },
  menus: [
    { name: "Gestión", minName: "Ges", id: 1, icon: "icon-flow-merge",
      options: [
        { name: "Empresas", route: "/admin/empresas", icon: "icon-black-tie",
        },
        { name: "Parámetros", route: "/admin/parametros", icon: "icon-cog",
        },
        { name: "Usuarios", route: "/admin/usuarios", icon: "icon-adult",
        },
        { name: "Perfiles & Accesos", route: "/admin/perfiles-accesos",
          icon: "icon-shield",
        },
        { name: "Backups", route: "/admin/backups", icon: "icon-database"
        },
      ]
    },
    { name: "Operaciones", minName: "Ope",  id: 2, icon: "icon-cube",
      options: [
        { name: "Sedes & Almacenes", route: "/operaciones/sedes-almacenes",
          icon: "icon-home-1"
        },
        { name: "Productos", route: "/operaciones/productos",
          icon: "icon-cube"
        },
        { name: "Productos Stock", route: "/operaciones/productos-stock",
          icon: "icon-chart-bar"
        },
        { name: "Almacén Movimientos", route: "/operaciones/almacen-movimientos",
          icon: "icon-truck"
        },
        { name: "Cajas & Bancos", route: "/operaciones/cajas",
          icon: "icon-suitcase"
        },
        { name: "Cajas Movimientos", route: "/operaciones/cajas-movimientos",
          icon: "icon-exchange"
        },
      ]  
    },
    { name: "Comercial", minName: "Com",  id: 3, icon: "icon-tasks",
      options: [
        { name: "Ventas", route: "/comercial/ventas",
          icon: "icon-flash"
        },
      ]  
    },
    { name: "UI Components", minName: "UIC",  id: 4, icon: "icon-buffer",
      options: [
        { name: "Test Table", route: "/develop-ui/test-table"
        },
        { name: "Test Cards", route: "/develop-ui/test-cards"
        },
        { name: "Chart Demo 1", route: "/develop-ui/chart-test"
        },
        { name: "API Test", route: "/develop-ui/api-test"
      },
      ]

    },
    { name: "CMS", minName: "CMS",  id: 5, icon: "icon-buffer",
      options: [
        { name: "Inicio", route: "/cms/page-1"
        },
        { name: "Nosotros", route: "/cms/page-2"
        },
        { name: "Tienda", route: "/cms/page-3"
        },
        { name: "Producto", route: "/cms/page-4"
        },
        { name: "Carrito Compra", route: "/cms/page-5"
        }, 
      ]  
    },
    { name: "Reportes", minName: "Rep",  id: 6, icon: "icon-tasks",
      options: [
        { name: "Productos",
        },
        { name: "Almacenes",
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