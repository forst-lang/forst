// Type definitions
export interface T_NGUd7jXQkPf {
  user: any;
  created_at: any;
}

export interface T_JrLVDGdHxf2 {
  name: any;
  age: any;
  email: any;
  id: any;
}

export interface User {
  id: any;
  name: any;
  age: any;
  email: any;
}

export interface CreateUserRequest {
  age: any;
  email: any;
  name: any;
}

export interface CreateUserResponse {
  user: User;
  created_at: any;
}

export interface T_TndA4HRxw7P {
  email: any;
  id: any;
  name: any;
  age: any;
}

export interface T_N142uWz7bTR {
  id: any;
  name: any;
  age: any;
  email: any;
}

// Function declarations
export function CreateUser(input: CreateUserRequest): Promise<void>;

export function GetUserById(id: any): Promise<void>;

export function UpdateUserAge(id: any, newAge: any): Promise<void>;

// Client extensions
declare module '@forst/client' {
  interface ForstClient {
    user: {
      CreateUser(input: CreateUserRequest): (void) => Promise<
      GetUserById(id: any): (void) => Promise<
      UpdateUserAge(id: any, newAge: any): (void) => Promise<
    };
  }
}
