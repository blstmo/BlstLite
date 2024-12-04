// components/VPSList.tsx
import { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Clock, Server, HardDrive } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';

const StatusBadge = ({ status }: { status: string }) => {
  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'bg-green-500/15 text-green-700 border-green-600/20';
      case 'creating':
        return 'bg-blue-500/15 text-blue-700 border-blue-600/20';
      case 'stopped':
        return 'bg-red-500/15 text-red-700 border-red-600/20';
      default:
        return 'bg-gray-500/15 text-gray-700 border-gray-600/20';
    }
  };

  return (
    <Badge 
      variant="outline" 
      className={`${getStatusColor(status)} font-medium capitalize`}
    >
      {status}
    </Badge>
  );
};

export default function VPSList() {
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
    return (
      <div className="space-y-4">
        {[1, 2].map((i) => (
          <Card key={i}>
            <CardContent className="py-4">
              <div className="flex items-start justify-between">
                <div className="space-y-3">
                  <Skeleton className="h-4 w-[200px]" />
                  <Skeleton className="h-4 w-[150px]" />
                </div>
                <Skeleton className="h-6 w-[100px]" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (instances.length === 0) {
    return (
      <div className="text-center py-8">
        <Server className="mx-auto h-12 w-12 text-gray-400" />
        <h3 className="mt-2 text-sm font-medium text-gray-900">No instances</h3>
        <p className="mt-1 text-sm text-gray-500">
          No active VPS instances found
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {instances.map((vps) => (
        <Card key={vps.id}>
          <CardContent className="py-4">
            <div className="flex items-start justify-between">
              <div className="space-y-3">
                <div className="flex items-center space-x-2">
                  <h3 className="font-semibold text-lg">{vps.name}</h3>
                  <StatusBadge status={vps.status} />
                </div>
                
                <div className="space-y-2 text-sm text-gray-500">
                  <div className="flex items-center gap-x-2">
                    <HardDrive className="h-4 w-4" />
                    <span className="capitalize">{vps.image_type}</span>
                  </div>
                  
                  <div className="flex items-center gap-x-2">
                    <Clock className="h-4 w-4" />
                    <span>{vps.time_remaining}</span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}