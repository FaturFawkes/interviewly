import "next-auth";
import "next-auth/jwt";

declare module "next-auth" {
  interface User {
    backendAccessToken?: string;
  }

  interface Session {
    backendAccessToken?: string;
    authError?: string;
    user: {
      id?: string;
      name?: string | null;
      email?: string | null;
      image?: string | null;
    };
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    backendAccessToken?: string;
    userId?: string;
    authError?: string;
  }
}
