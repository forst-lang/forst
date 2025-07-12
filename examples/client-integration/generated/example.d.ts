// Type definitions
export interface ICreateUserRequest {
  name: any;
  age: any;
  email: any;
}

export interface ICreateUserResponse {
  user: any;
  created_at: any;
}

export interface IT_NGUd7jXQkPf {
  user: any;
  created_at: any;
}

export interface IT_JrLVDGdHxf2 {
  id: any;
  name: any;
  age: any;
  email: any;
}

export interface IT_TndA4HRxw7P {
  id: any;
  name: any;
  age: any;
  email: any;
}

export interface IT_N142uWz7bTR {
  id: any;
  name: any;
  age: any;
  email: any;
}

export interface IUser {
  id: any;
  name: any;
  age: any;
  email: any;
}

// Function declarations
export function CreateUser(input: ICreateUserRequest): Promise<void>;

export function GetUserById(id: any): Promise<void>;

export function UpdateUserAge(id: any, newAge: any): Promise<void>;

// Client extensions
declare module '@forst/sidecar' {
  interface ForstClient {
    CreateUser(input: ICreateUserRequest): Promise<void>;
    GetUserById(id: any): Promise<void>;
    UpdateUserAge(id: any, newAge: any): Promise<void>;
  }
}
