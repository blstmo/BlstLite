// app/api/vps/list/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function GET() {
  try {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/list`, {
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
    console.error('VPS list error:', error);
    return NextResponse.json(
      { error: 'Failed to fetch VPS list' },
      { status: 500 }
    );
  }
}
