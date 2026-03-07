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
