import { withAuth } from "next-auth/middleware";

export default withAuth({
  callbacks: {
    authorized: ({ token }) => !!token,
  },
  pages: {
    signIn: "/auth/sign-in",
  },
});

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/upload/:path*",
    "/practice/:path*",
    "/analytics/:path*",
  ],
};
