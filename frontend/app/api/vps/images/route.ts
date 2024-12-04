// app/api/vps/images/route.ts
import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function GET() {
  try {
    // Log the request URL and headers for debugging
    console.log('Fetching images from:', `${API_CONFIG.baseUrl}/api/images/list`);
    console.log('Using API Key:', API_CONFIG.apiKey ? 'Present' : 'Missing');

    const response = await fetch(`${API_CONFIG.baseUrl}/api/images/list`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey || '',
        'Accept': 'application/json',
      },
      // Add cache: 'no-store' to prevent caching
      cache: 'no-store'
    });

    if (!response.ok) {
      console.error('Backend response not OK:', response.status, response.statusText);
      const errorText = await response.text();
      console.error('Error response body:', errorText);
      throw new Error(`Backend responded with status ${response.status}`);
    }

    const data = await response.json();

    // Log the received data
    console.log('Received image data:', data);

    // Ensure we're returning an array
    if (!Array.isArray(data)) {
      console.error('Received non-array data:', data);
      throw new Error('Backend returned invalid data format');
    }

    // Return the array as JSON response
    return NextResponse.json(data, {
      headers: {
        'Content-Type': 'application/json',
      }
    });
  } catch (error) {
    console.error('Error in /api/vps/images:', error);
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to fetch images' },
      { status: 500 }
    );
  }
}