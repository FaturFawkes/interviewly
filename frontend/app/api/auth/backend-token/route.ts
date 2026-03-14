import { NextResponse } from "next/server";
import { getServerSession } from "next-auth";

import { authOptions } from "@/lib/auth/options";

export async function GET() {
  const session = await getServerSession(authOptions);

  if (!session?.backendAccessToken) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  return NextResponse.json({ access_token: session.backendAccessToken });
}
