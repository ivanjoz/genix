export interface IUser {
  ID: number;
  CompanyID: number;
  FirstName: string;
  LastName: string;
  Email: string;
  Usuario: string;
  DocumentNumber: string;
  JobTitle: string;
  ProfileIDs: number[];
  AccessLevelIDs: number[];
  Status: number;
  Updated: number;
  Created: number;
  Password: string;
  Password2: string;
  CreatedBy: number;
  UpdatedBy: number;
  CacheVersion: number;
  PasswordHash: string;
}

export interface IProfile {
  ID: number;
  CompanyID: number;
  Name: string;
  Description?: string;
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
  AccesosComputed: string;
  TokenExpTime: number;
  CompanyID: number;
}
