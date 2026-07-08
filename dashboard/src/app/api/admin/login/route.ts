import { NextResponse } from 'next/server';

export async function POST(request: Request) {
  try {
    const body = await request.text();
    const res = await fetch('http://127.0.0.1:8080/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    });
    const text = await res.text();
    return new NextResponse(text, {
      status: res.status,
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error: any) {
    console.error("Next.js proxy error (login):", error.message);
    return NextResponse.json({ error: "Proxy connection failed" }, { status: 500 });
  }
}
