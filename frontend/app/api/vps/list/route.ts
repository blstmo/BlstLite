// app/api/vps/list/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';
import type { VPSPublic } from '@/types/vps';

function calculateTimeRemaining(expiresAt: string): string {
  const remaining = new Date(expiresAt).getTime() - Date.now();
  const minutes = Math.max(0, Math.floor(remaining / (1000 * 60)));
  
  if (minutes === 0) return 'Expired';
  return `${minutes} minutes remaining`;
}

function sanitizeVPSData(vps: VPSBackend): VPSPublic {
  return {
    id: vps.id,
    name: vps.name,
    status: vps.status,
    image_type: vps.image_type,
    created_at: new Date(vps.created_at).toISOString(),
    expires_at: new Date(vps.expires_at).toISOString(),
    time_remaining: calculateTimeRemaining(vps.expires_at),
  };
}

export async function GET() {
  try {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/list`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey!,
        'Accept': 'application/json',
      },
      cache: 'no-store',
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Backend error (${response.status}): ${errorText}`);
    }

    const data = await response.json() as VPSBackend[];

    if (!Array.isArray(data)) {
      throw new Error('Invalid response format: expected an array');
    }

    // Remove sensitive data before sending to client
    const sanitizedData: VPSPublic[] = data.map(sanitizeVPSData);

    return NextResponse.json(sanitizedData, {
      headers: {
        'Cache-Control': 'no-store, must-revalidate',
        'Content-Type': 'application/json',
      },
    });
  } catch (error) {
    console.error('VPS list error:', error);
    
    const message = error instanceof Error 
      ? error.message
      : 'Failed to fetch VPS list';

    return NextResponse.json(
      { error: message },
      { 
        status: 500,
        headers: {
          'Cache-Control': 'no-store, must-revalidate',
          'Content-Type': 'application/json',
        },
      }
    );
  }
}