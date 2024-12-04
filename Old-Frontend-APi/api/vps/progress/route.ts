// app/api/vps/progress/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const id = searchParams.get('id');

    if (!id) {
      return NextResponse.json(
        { error: 'VPS ID is required' },
        { status: 400 }
      );
    }

    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/progress?id=${id}`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });

    if (!response.ok) {
      // If the VPS is not found, forward the 404 status
      if (response.status === 404) {
        return NextResponse.json(
          { error: 'VPS not found' },
          { status: 404 }
        );
      }
      throw new Error(`Backend error: ${response.status}`);
    }

    const progressData = await response.json();

    return NextResponse.json(progressData);
  } catch (error) {
    console.error('VPS progress check error:', error);
    return NextResponse.json(
      { error: 'Failed to check VPS progress' },
      { status: 500 }
    );
  }
}

// Types for the progress response
export interface VPSProgressResponse {
  stage: string;
  progress: number;
  status: string;
  error?: string;
}