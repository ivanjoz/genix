export interface IUsuario {
  id: number;
  companyID: number;
  nombres: string;
  apellidos: string;
  email: string;
  usuario: string;
  documentoNro: string;
  cargo: string;
  perfilesIDs: number[];
  rolesIDs: number[];
  ss: number;
  upd: number;
  created: number;
  password1: string;
  password2: string;
  //Extra
  accesosIDs: number[];
}

export interface IPerfil {
  id: number;
  nombre: string;
  descripcion?: string;
  accesos: number[];
  modulosIDs: number[];
  accesosMap: Map<number, number[]>;
  ss: number;
  upd: number;
}

export interface IImageResult {
  id: number;
  imageName: string;
  description?: string;
}

export interface ILoginResult {
  UserID: number;
  UserNames: string;
  UserEmail: string;
  UserToken: string;
  UserInfo: string;
  TokenExpTime: number;
  EmpresaID: number;
}
