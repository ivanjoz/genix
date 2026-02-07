export interface IInput {
  id?: number;
  saveOn: any;
  save: string;
  label?: string;
  css?: string;
  inputCss?: string;
  required?: boolean;
  validator?: (v: string | number) => boolean;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  onChange?: () => void;
  postValue?: any;
  baseDecimals?: number;
  content?: string | any;
  transform?: (v: string | number) => string | number;
  useTextArea?: boolean;
  rows?: number;
}

// Menu Types
export interface IMenuRecord {
	name: string;
	minName?: string;
	id?: number;
	route?: string;
	options?: IMenuRecord[];
	icon?: string;
	onlySaaS?: boolean
	descripcion?: string
	pageTabs?: string[]
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
			name: 'Configuración',
			minName: 'CON',
			icon: 'icon-cog',
			options: [
				{
					name: 'Empresas',
					minName: 'EMP',
					route: '/configuracion/empresas',
					icon: 'icon-building'
				},
				{
					name: 'Usuarios',
					minName: 'USR',
					route: '/configuracion/usuarios',
					icon: 'icon-users'
				},
				{ name: 'Sedes', minName: 'SED', route: '/negocio/sedes-almacenes', icon: 'icon-location' }
			]
		},
		{
			id: 2,
			name: 'Negocio',
			minName: 'NEG',
			icon: 'icon-box',
			options: [
				{
					name: 'Productos',
					minName: 'PRO',
					route: '/negocio/productos',
					icon: 'icon-cube'
				},
				{
					name: 'Almacenes',
					minName: 'ALM',
					route: '/negocio/sedes-almacenes',
					icon: 'icon-archive'
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
					route: '/negocio/clientes-proveedores',
					icon: 'icon-user'
				},
				{
					name: 'Reportes',
					minName: 'REP',
					route: '/comercial/gestion-ventas',
					icon: 'icon-chart-bar'
				}
			]
		}
	]
};
