import { NextResponse } from 'next/server';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const tenant_id = searchParams.get('tenant_id');
  const authHeader = request.headers.get('Authorization');

  try {
    const res = await fetch(`http://127.0.0.1:8080/admin/metrics?tenant_id=${tenant_id}`, {
      headers: authHeader ? { Authorization: authHeader } : {},
    });
    const text = await res.text();
    return new NextResponse(text, {
      status: res.status,
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error: any) {
    console.error("Next.js proxy error (metrics):", error.message);
    return NextResponse.json({ error: "Proxy connection failed" }, { status: 500 });
  }
}
