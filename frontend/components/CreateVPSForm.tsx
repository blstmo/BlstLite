import React, { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card } from '@/components/ui/card';

interface CreateVPSFormProps {
  onSuccess: (vps: VPS) => void;
}

interface OSImage {
  id: string;
  name: string;
  imagePath: string;
}

export default function CreateVPSForm({ onSuccess }: CreateVPSFormProps) {
  const [name, setName] = useState('');
  const [selectedImage, setSelectedImage] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [availableImages, setAvailableImages] = useState<OSImage[]>([]);

  // Function to generate image path and display name based on OS ID
  const getOSDetails = (osId: string): OSImage => {
    const displayNames: Record<string, string> = {
      'ubuntu-22.04': 'Ubuntu 22.04',
      'debian-11': 'Debian 11',
      'fedora-38': 'Fedora 38',
      'arch-linux': 'Arch Linux'
    };

    // Extract the base OS name (e.g., 'ubuntu' from 'ubuntu-22.04')
    const baseOsName = osId.split('-')[0];

    return {
      id: osId,
      name: displayNames[osId] || osId,
      imagePath: `/images/os/${baseOsName}.png`
    };
  };

  useEffect(() => {
    const fetchImages = async () => {
      try {
        const response = await fetch('/api/vps/images');
        if (!response.ok) throw new Error('Failed to fetch available images');
        const images = await response.json();
        const formattedImages = images.map(getOSDetails);
        setAvailableImages(formattedImages);
        if (formattedImages.length > 0) {
          setSelectedImage(formattedImages[0].id);
        }
      } catch (err) {
        console.error('Failed to fetch images:', err);
        setError('Failed to load available images');
      }
    };

    fetchImages();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const response = await fetch('/api/vps/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, image_type: selectedImage }),
      });

      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || 'Failed to create VPS');
      }

      onSuccess(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create VPS');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor="name">Instance Name</Label>
        <Input
          id="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="my-ubuntu-vps"
          required
          disabled={loading}
          minLength={3}
          maxLength={50}
          pattern="[a-zA-Z0-9-]+"
        />
      </div>

      <div className="space-y-2">
        <Label>Select Operating System</Label>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {availableImages.map((os) => (
            <Card
              key={os.id}
              className={`p-4 cursor-pointer transition-all hover:ring-2 hover:ring-primary ${
                selectedImage === os.id ? 'ring-2 ring-primary bg-primary/5' : ''
              }`}
              onClick={() => setSelectedImage(os.id)}
            >
              <div className="flex flex-col items-center space-y-2">
                <div className="w-16 h-16 relative">
                  <img
                    src={os.imagePath}
                    alt={os.name}
                    className="w-full h-full object-contain"
                    onError={(e) => {
                      const img = e.target as HTMLImageElement;
                      img.src = '/images/os/default.png';
                    }}
                  />
                </div>
                <span className="text-sm font-medium text-center">{os.name}</span>
              </div>
            </Card>
          ))}
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" disabled={loading} className="w-full">
        {loading ? 'Creating...' : 'Create VPS'}
      </Button>
    </form>
  );
}