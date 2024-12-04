// app/actions.ts
'use server'

import { API_CONFIG } from '@/lib/config';
import { VPS, ResourceMetrics  } from '@/types/vps'

interface CreateVPSParams {
  name: string;
  hostname: string;
  image_type: string;
}

export async function getVPSList(): Promise<VPS[]> {
  console.log('Fetching VPS list...');

  const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/list`, {
    headers: {
      'X-API-Key': API_CONFIG.apiKey!
    },
    next: { revalidate: 0 } // Disable caching
  });

  console.log('Received response:', response);

  if (!response.ok) {
    console.error('Failed to fetch VPS list. Status:', response.status);
    throw new Error('Failed to fetch VPS list');
  }

  const vpsList = await response.json();
  console.log('VPS list fetched successfully:', vpsList);

  return vpsList;
}

  
  export async function deleteVPS(id: string) {
    console.log(`Attempting to delete VPS with ID: ${id}`);
  
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/delete?id=${id}`, {
      method: 'DELETE',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });
  
    console.log('Received response:', response);
  
    if (!response.ok) {
      console.error('Failed to delete VPS. Status:', response.status);
      throw new Error('Failed to delete VPS');
    }
  
    const responseData = await response.json();
    console.log('VPS delete successful, response data:', responseData);
  
    return responseData;
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
      if (response.status === 409) {
        throw new Error('You already have an active VPS. Only one VPS per IP address is allowed.');
      }
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
export async function startVPS(id: string) {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/start?id=${id}`, {
      method: 'POST',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });
  
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Failed to start VPS');
    }
  
    return response.ok;
  }
  
  export async function stopVPS(id: string) {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/stop?id=${id}`, {
      method: 'POST',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });
  
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Failed to stop VPS');
    }
  
    return response.ok;
  }
  

export async function restartVPS(id: string) {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/restart?id=${id}`, {
      method: 'POST',
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      }
    });
  
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Failed to restart VPS');
    }
  
    return response.ok;
  }

  export async function getVPSMetrics(id: string): Promise<ResourceMetrics[]> {
    const response = await fetch(`${API_CONFIG.baseUrl}/api/vps/metrics?id=${id}`, {
      headers: {
        'X-API-Key': API_CONFIG.apiKey!
      },
      next: { revalidate: 0 } // Disable caching
    });
  
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Failed to fetch VPS metrics');
    }
  
    return response.json();
  }
  

