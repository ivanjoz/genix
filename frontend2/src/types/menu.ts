// Menu Types

export interface IMenuRecord {
	name: string;
	minName?: string;
	id?: number;
	route?: string;
	options?: IMenuRecord[];
	icon?: string;
}

export interface IModule {
	id: number;
	name: string;
	menus: IMenuRecord[];
}

// Default module configuration
export const defaultModule: IModule = {
	id: 1,
	name: 'Sistema',
	menus: [
		{
			id: 1,
			name: 'Administración',
			minName: 'ADM',
			icon: 'icon-cog',
			options: [
				{
					name: 'Empresas',
					minName: 'EMP',
					route: '/admin/empresas',
					icon: 'icon-building'
				},
				{
					name: 'Usuarios',
					minName: 'USR',
					route: '/admin/usuarios',
					icon: 'icon-users'
				},
				{ name: 'Sedes', minName: 'SED', route: '/admin/sedes', icon: 'icon-location' }
			]
		},
		{
			id: 2,
			name: 'Operaciones',
			minName: 'OPE',
			icon: 'icon-box',
			options: [
				{
					name: 'Productos',
					minName: 'PRO',
					route: '/operaciones/productos',
					icon: 'icon-cube'
				},
				{
					name: 'Almacenes',
					minName: 'ALM',
					route: '/operaciones/almacenes',
					icon: 'icon-archive'
				},
				{
					name: 'Movimientos',
					minName: 'MOV',
					route: '/operaciones/movimientos',
					icon: 'icon-docs'
				}
			]
		},
		{
			id: 3,
			name: 'Comercial',
			minName: 'COM',
			icon: 'icon-chart-line',
			options: [
				{
					name: 'Ventas',
					minName: 'VEN',
					route: '/comercial/ventas',
					icon: 'icon-basket'
				},
				{
					name: 'Clientes',
					minName: 'CLI',
					route: '/comercial/clientes',
					icon: 'icon-user'
				},
				{
					name: 'Reportes',
					minName: 'REP',
					route: '/comercial/reportes',
					icon: 'icon-chart-bar'
				}
			]
		}
	]
};

