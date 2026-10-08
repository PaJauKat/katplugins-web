import { NextResponse, type NextRequest } from "next/server";

// Chequeo ligero de sesion solo para UX (redirigir a /login).
// La validacion real de la sesion la hace el backend Go en cada endpoint.
const SESSION_COOKIE = "kat_session";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isProtected =
    pathname.startsWith("/dashboard") || pathname.startsWith("/admin");

  if (isProtected && !request.cookies.get(SESSION_COOKIE)) {
    const url = request.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  return NextResponse.next({ request });
}

export const config = {
  matcher: ["/dashboard/:path*", "/admin/:path*"],
};
