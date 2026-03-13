export interface IUsuario {
  ID: number;
  EmpresaID: number;
  Nombres: string;
  Apellidos: string;
  Email: string;
  Usuario: string;
  DocumentoNro: string;
  Cargo: string;
  PerfilesIDs: number[];
  AccesosNivelIDs: number[];
  RolesIDs: number[];
  Status: number;
  Updated: number;
  Created: number;
  Password: string;
  Password2: string;
  CreatedBy: number;
  UpdatedBy: number;
  CacheVersion: number;
  PasswordHash: string;
  //Extra
  AccesosIDs: number[];
}

export interface IPerfil {
  ID: number;
  EmpresaID: number;
  Nombre: string;
  Descripcion?: string;
  Accesos: number[];
  Modulos: number[];
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
