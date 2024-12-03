// components/VPSList.tsx
import { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Clock } from 'lucide-react';

interface VPSListProps {
  onSelect: (vps: VPS) => void;
}

export default function VPSList({ onSelect }: VPSListProps) {
  const [instances, setInstances] = useState<VPS[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchInstances = async () => {
    try {
      const response = await fetch('/api/vps/list');
      if (response.ok) {
        const data = await response.json();
        setInstances(data);
      }
    } catch (error) {
      console.error('Failed to fetch VPS instances:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstances();
    const interval = setInterval(fetchInstances, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (instances.length === 0) {
    return <div className="text-gray-500">No active instances</div>;
  }

  return (
    <div className="space-y-4">
      {instances.map((vps) => (
        <div
          key={vps.id}
          className="border rounded-lg p-4 hover:border-blue-500 transition-colors"
        >
          <div className="flex justify-between items-start">
            <div>
              <h3 className="font-bold">{vps.name}</h3>
              <div className="text-sm text-gray-500 space-y-1">
                <p>Status: {vps.status}</p>
                <p className="flex items-center">
                  <Clock className="h-4 w-4 mr-1" />
                  Expires: {new Date(vps.expires_at).toLocaleTimeString()}
                </p>
              </div>
            </div>
            <Button onClick={() => onSelect(vps)}>
              Connect
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}