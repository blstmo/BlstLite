'use client'

import { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Clock, Server, HardDrive, Trash2, Loader2 } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { getVPSList, deleteVPS } from '../app/actions';

const StatusBadge = ({ status }: { status: string }) => {
  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'bg-green-500/15 text-green-700 border-green-600/20';
      case 'creating':
        return 'bg-blue-500/15 text-blue-700 border-blue-600/20';
      case 'stopped':
        return 'bg-red-500/15 text-red-700 border-red-600/20';
      case 'failed':
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

const formatTimeRemaining = (expiresAt: string) => {
  const now = new Date();
  const expiry = new Date(expiresAt);
  const diff = expiry.getTime() - now.getTime();
  
  if (diff <= 0) return 'Expired';
  
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'Less than a minute';
  if (minutes === 1) return '1 minute remaining';
  return `${minutes} minutes remaining`;
};

const getOSIcon = (imageType: string): string => {
  const os = imageType.split('-')[0].toLowerCase();
  return `/images/os/${os}.png`;
};

export default function VPSList() {
  const [instances, setInstances] = useState<VPS[]>([]);
  const [loading, setLoading] = useState(true);
  const [deletingIds, setDeletingIds] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string | null>(null);

  const fetchInstances = async () => {
    try {
      const data = await getVPSList();
      setInstances(data);
      setError(null);
    } catch (error) {
      console.error('Failed to fetch VPS instances:', error);
      setError('Failed to load VPS instances');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstances();
    const interval = setInterval(fetchInstances, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleDelete = async (id: string) => {
    try {
      setDeletingIds(prev => new Set([...prev, id]));
      await deleteVPS(id);
      setInstances(prev => prev.filter(instance => instance.id !== id));
      setError(null);
    } catch (error) {
      console.error('Failed to delete VPS:', error);
      setError('Failed to delete VPS');
    } finally {
      setDeletingIds(prev => {
        const newSet = new Set(prev);
        newSet.delete(id);
        return newSet;
      });
    }
  };

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

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {instances.length === 0 && !error ? (
        <div className="text-center py-8">
          <Server className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No instances</h3>
          <p className="mt-1 text-sm text-gray-500">
            No active VPS instances found
          </p>
        </div>
      ) : (
        instances.map((vps) => (
          <Card key={vps.id} className="relative overflow-hidden">
            {vps.status === 'creating' && (
              <div className="absolute bottom-0 left-0 right-0 h-1 bg-primary/10">
                <div className="h-full bg-primary animate-pulse" style={{ width: '100%' }} />
              </div>
            )}
            <CardContent className="py-4">
              <div className="flex items-start justify-between">
                <div className="space-y-3">
                  <div className="flex items-center space-x-2">
                    <h3 className="font-semibold text-lg">{vps.name}</h3>
                    <StatusBadge status={vps.status} />
                  </div>
                  
                  <div className="space-y-2 text-sm text-gray-500">
                    <div className="flex items-center gap-x-2">
                      <div className="w-4 h-4 relative flex-shrink-0">
                        <img
                          src={getOSIcon(vps.image_type)}
                          alt={vps.image_type}
                          className="object-contain"
                          onError={(e) => {
                            const img = e.target as HTMLImageElement;
                            img.src = '/images/os/default.png';
                          }}
                        />
                      </div>
                      <span className="capitalize">{vps.image_type}</span>
                    </div>
                    
                    <div className="flex items-center gap-x-2">
                      <Clock className="h-4 w-4" />
                      <span>{formatTimeRemaining(vps.expires_at)}</span>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  );
}