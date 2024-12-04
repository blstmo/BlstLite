export interface VPS {
  id: string;
  name: string;
  hostname: string;
  status: string;
  vnc_port: number;
  created_at: string;
  expires_at: string;
  password: string;
}

export interface VPSPublic {
  name: string;
  hostname: string;
  status: string;
  image_type: string;
  created_at: string;
  expires_at: string;
  time_remaining: string; // Human readable time remaining until expiry
}

export interface VPSBackend {
  id: string;
  name: string;
  hostname: string;
  status: string;
  image_type: string;
  qemu_pid?: number;
  vnc_port: number;
  ssh_port: number;
  created_at: string;
  expires_at: string;
  image_path: string;
  password: string;
}