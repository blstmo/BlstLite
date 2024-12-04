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

export interface ResourceMetrics {
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  disk: DiskMetrics;
  network: NetworkMetrics;
  time: string;
}

export interface CPUMetrics {
  usage: number;  // Percentage (0-100)
}

export interface MemoryMetrics {
  used: number;   // Bytes
  total: number;  // Bytes
  cache: number;  // Bytes
}

export interface DiskMetrics {
  read_bytes: number;
  write_bytes: number;
  read_ops: number;
  write_ops: number;
  read_speed: number;  // Bytes per second
  write_speed: number; // Bytes per second
}

export interface NetworkMetrics {
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_speed: number; // Bytes per second
  tx_speed: number; // Bytes per second
}