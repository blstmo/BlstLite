'use client';

import { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import CreateVPSForm from '@/components/CreateVPSForm';
import VPSList from '@/components/VPSList';
import VPSDetail from '@/components/VPSDetail';
import { Card, CardContent } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Clock } from 'lucide-react';

export default function Home() {
  const [selectedVPS, setSelectedVPS] = useState<VPS | null>(null);

  return (
    <main className="container mx-auto p-4 space-y-8">
      <h1 className="text-4xl font-bold mb-8">Temporary VPS Service</h1>
      
      <Alert>
        <Clock className="h-4 w-4" />
        <AlertDescription>
          Each VPS instance runs for 15 minutes with 4GB RAM, 50GB storage, 
          and 50Mbps/15Mbps network speed.
        </AlertDescription>
      </Alert>

      {selectedVPS ? (
        <VPSDetail vps={selectedVPS} onClose={() => setSelectedVPS(null)} />
      ) : (
        <div className="grid md:grid-cols-2 gap-8">
          <Card>
            <CardContent className="p-6">
              <h2 className="text-2xl font-bold mb-4">Create New VPS</h2>
              <CreateVPSForm onSuccess={setSelectedVPS} />
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="p-6">
              <h2 className="text-2xl font-bold mb-4">Active Instances</h2>
              <VPSList onSelect={setSelectedVPS} />
            </CardContent>
          </Card>
        </div>
      )}
    </main>
  );
}
