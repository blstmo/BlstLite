// components/CreateVPSForm.tsx
import { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface CreateVPSFormProps {
  onSuccess: (vps: VPS) => void;
}

export default function CreateVPSForm({ onSuccess }: CreateVPSFormProps) {
  const [name, setName] = useState('');
  const [imageType, setImageType] = useState('ubuntu-22.04');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [availableImages, setAvailableImages] = useState<string[]>([]);

  useEffect(() => {
    const fetchImages = async () => {
      try {
        const response = await fetch('/api/vps/images');
        if (!response.ok) {
          throw new Error('Failed to fetch available images');
        }
        const images = await response.json();
        setAvailableImages(images);
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
      console.log('Submitting VPS creation:', { name, imageType });

      const response = await fetch('/api/vps/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, image_type: imageType }),
      });

      const data = await response.json();
      console.log('Create VPS response:', data);

      if (!response.ok) {
        throw new Error(data.error || 'Failed to create VPS');
      }

      onSuccess(data);
    } catch (err) {
      console.error('VPS creation error:', err);
      setError(err instanceof Error ? err.message : 'Failed to create VPS');
    } finally {
      setLoading(false);
    }
  };

  const getImageDisplayName = (image: string) => {
    const nameMap: Record<string, string> = {
      'ubuntu-22.04': 'Ubuntu 22.04',
      'debian-11': 'Debian 11',
      'fedora-38': 'Fedora 38',
      'arch-linux': 'Arch Linux'
    };
    return nameMap[image] || image;
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
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
        <Label htmlFor="imageType">Operating System</Label>
        <Select
          value={imageType}
          onValueChange={setImageType}
          disabled={loading}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Select an operating system" />
          </SelectTrigger>
          <SelectContent>
            {availableImages.map((image) => (
              <SelectItem key={image} value={image}>
                {getImageDisplayName(image)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
