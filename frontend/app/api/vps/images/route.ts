// app/api/vps/images/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function GET() {
  try {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/images/list`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey || ''
      }
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch images: ${response.status}`);
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching images:', error);
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to fetch images' },
      { status: 500 }
    );
  }
}