export interface IUser {
  ID: number;
  CompanyID: number;
  FirstName: string;
  LastName: string;
  Email: string;
  User: string;
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
  UpdatedVersion: number;
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

export interface ILoginResult {
  UserID: number;
  UserNames: string;
  UserEmail: string;
  UserToken: string;
  UserInfo: string;
  AccesosComputed: string;
  TokenExpTime: number;
  CompanyID: number;
  // The company has no warehouse or no cash bank yet, so it cannot operate: the login routes to
  // the "Datos Iniciales" page instead of home.
  InitialDataPending?: boolean;
}
