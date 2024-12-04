// app/actions.ts
'use server'

import { API_CONFIG } from '@/lib/config';
import { VPS } from '@/types/vps'

interface CreateVPSParams {
  name: string;
  hostname: string;
  image_type: string;
}

export async function getVPSList(): Promise<VPS[]> {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/list`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      },
      next: { revalidate: 0 } // Disable caching
    });
  
    if (!response.ok) {
      throw new Error('Failed to fetch VPS list');
    }
  
    return response.json();
  }
  
  export async function deleteVPS(id: string) {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/delete?id=${id}`, {
      method: 'DELETE',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });
  
    if (!response.ok) {
      throw new Error('Failed to delete VPS');
    }
  
    return response.json();
  }

export async function createVPS(params: CreateVPSParams) {
  const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/create`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_CONFIG.apiKey!
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to create VPS');
  }

  return response.json();
}

export async function checkVPSProgress(id: string) {
  const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/progress?id=${id}`, {
    headers: {
      'X-API-Key': API_CONFIG.apiKey!
    }
  });

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to check VPS progress');
  }

  return response.json();
}

export async function getVPSDetails(id: string) {
  const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/get?id=${id}`, {
    headers: {
      'X-API-Key': API_CONFIG.apiKey!
    }
  });

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get VPS details');
  }

  return response.json();
}

export async function getAvailableImages() {
  const response = await fetch(`${API_CONFIG.baseUrl}/api/images/list`, {
    headers: {
      'X-API-Key': API_CONFIG.apiKey!
    }
  });

  if (!response.ok) {
    throw new Error('Failed to fetch available images');
  }

  return response.json();
}