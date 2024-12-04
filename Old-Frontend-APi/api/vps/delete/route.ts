// app/api/vps/delete/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function DELETE(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const id = searchParams.get('id');

    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/delete?id=${id}`, {
      method: 'DELETE',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });

    if (!response.ok) {
      throw new Error(`Backend error: ${response.status}`);
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('VPS deletion error:', error);
    return NextResponse.json(
      { error: 'Failed to delete VPS' },
      { status: 500 }
    );
  }
}