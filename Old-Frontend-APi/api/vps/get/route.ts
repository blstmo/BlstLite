// app/api/vps/get/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const id = searchParams.get('id');

    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/get?id=${id}`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });

    if (!response.ok) {
      throw new Error(`Backend error: ${response.status}`);
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('VPS get error:', error);
    return NextResponse.json(
      { error: 'Failed to fetch VPS details' },
      { status: 500 }
    );
  }
}