import { NextResponse } from 'next/server';
import { API_CONFIG } from '@/lib/config';

export async function POST(request: Request) {
  try {
    const body = await request.json();
    
    console.log('Attempting to create VPS with body:', body);
    console.log('Using API URL:', API_CONFIG.baseUrl);

    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': API_CONFIG.apiKey || ''
      },
      body: JSON.stringify({
        name: body.name,
        image_type: body.image_type || 'ubuntu-22.04' // Default to Ubuntu if not specified
      })
    });

    const responseText = await response.text();
    console.log('Backend response:', response.status, responseText);

    if (!response.ok) {
      throw new Error(`Backend error: ${response.status}. Response: ${responseText}`);
    }

    let data;
    try {
      data = JSON.parse(responseText);
    } catch (e) {
      console.error('Failed to parse JSON response:', responseText);
      throw new Error('Invalid JSON response from backend');
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error('Full VPS creation error:', error);
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to create VPS' },
      { status: 500 }
    );
  }
}