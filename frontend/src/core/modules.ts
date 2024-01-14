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
    usuarios: "id",
    seguridad_accesos: "id",
    perfiles: "id",
    fetchedIDBTables: 'key',
    cacheUpdates: 'key,route',
    cacheStatic: 'key',
    // Scylla
    productos: "ID",
    sedes_almacenes: '[_pk+ID]',  
    listas_compartidas: 'ID',
    pais_ciudades: '[PaisID+ID]', 
  },
  menus: [
    { name: "Gestión", minName: "Ges", id: 1, icon: "icon-flow-merge",
      options: [
        { name: "Empresas", route: "/admin/empresas"
        },
        { name: "Usuarios", route: "/admin/usuarios"
        },
        { name: "Perfiles & Accesos", route: "/admin/perfiles-accesos",
        },
      ]
    },
    { name: "Operaciones", minName: "Ope",  id: 2, icon: "icon-cube",
      options: [
        { name: "Sedes & Almacenes", route: "/operaciones/sedes-almacenes"
        },
        { name: "Productos", route: "/operaciones/productos"
        },
        { name: "Cajas", route: "/operaciones/cajas"
        },
      ]
  
    },
    { name: "UI Components", minName: "UIC",  id: 3, icon: "icon-buffer",
      options: [
        { name: "Test Table", route: "/develop-ui/test-table"
        },
        { name: "Chart Demo 1", route: "/develop-ui/chart-test"
        },
      ]

    },
    { name: "Reportes", minName: "Rep",  id: 4, icon: "icon-tasks",
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

  },
  menus: [
    { name: "Productos", minName: "PRD",  id: 2,
      options: [
        { name: "Módulo de Venta",
        },
        { name: "Productos",
        },
        { name: "Almacenes",
        },
      ]  
    },
  ]
}

const Modules: IModule[] = [AdminModule, ComercialModule]
export default Modules