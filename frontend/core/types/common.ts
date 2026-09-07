// One access as an operator granted it. A profile and a user grant access the same way, so both
// carry a list of these — mirrors backend coreTypes.AccesoGrantRecord.
export interface IAccesoGrant {
  AccesoID: number;
  // The single widest level granted on this access, 1..4.
  Nivel: number;
  // Sub-access ids; 1 is "Todos" and satisfies every check on the access. Absent for most accesses.
  SubAccesos?: number[];
}

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
  // Granted directly on this user, on top of whatever their profiles grant.
  AccesosGrants: IAccesoGrant[];
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
  AccesosGrants: IAccesoGrant[];
  Modulos: number[];
  // Editable form shape, stripped before the POST: accesoID -> nivel, and accesoID -> sub-access
  // ids. One nivel, not a list — every consumer collapses to the highest one anyway.
  accesosMap: Map<number, number>;
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
