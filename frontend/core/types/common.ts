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
  // accesoID * 100 + subAccesoID. Readable on purpose: the profile is what a human edits, so the
  // binary packing happens once, in the backend, when a user's grants are computed.
  SubAccesos: number[];
  Modulos: number[];
  accesosMap: Map<number, number[]>;
  // Editable form shape, stripped before the POST like accesosMap.
  subAccesosMap: Map<number, number[]>;
  ss: number;
  upd: number;
}

export interface ILoginResult {
  UserID: number;
  UserNames: string;
  UserEmail: string;
  UserToken: string;
  // Ciphered with the cipher key sent on login; UserInfoPlain replaces it when none was sent.
  UserInfo?: string;
  UserInfoPlain?: string;
  AccesosComputed: string;
  AccesosSubComputed: string;
  TokenExpTime: number;
  CompanyID: number;
  // The company has no warehouse or no cash bank yet, so it cannot operate: the login routes to
  // the "Datos Iniciales" page instead of home.
}
